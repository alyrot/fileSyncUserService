package main

import (
	"UserService/domain"
	"UserService/services/github.com/alyrot/UserServiceSchema"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"reflect"
	"testing"
)

//setupTestENV creates a client connected to an in memory service using an inmemory db
func setupTestENV(ctx context.Context) (UserServiceSchema.UserServiceClient, error) {
	//this will create an in memory sqlite db
	dsn := "file::memory:?cache=shared"
	db, err := SetupDB(dsn)
	if err != nil {
		return nil, err
	}

	//create server
	bufferSize := 1024 * 1024
	lis := bufconn.Listen(bufferSize)
	server := SetupGRPCServer(db)
	go func() {
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()
	go func() {
		<-ctx.Done()
		server.Stop()
		_ = lis.Close()
	}()

	conn, _ := grpc.Dial("", grpc.WithContextDialer(
		func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		},
	), grpc.WithInsecure())

	client := UserServiceSchema.NewUserServiceClient(conn)
	return client, err
}

func checkUserNoID(want *domain.User, got *UserServiceSchema.User) error {
	if want.Email != got.Email {
		return fmt.Errorf("want email %v got %v", want.Email, got.Email)
	}
	parsedPK, err := x509.ParsePKIXPublicKey(got.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key : %v", err)
	}
	if !want.PublicKey.Equal(parsedPK) {
		return fmt.Errorf("public keys do not match")
	}
	if !reflect.DeepEqual(want.WrappedPrivateKey, got.WrappedPrivateKey) {
		return fmt.Errorf("wanted wantWrappedPrivateKey %v got %v", want.WrappedPrivateKey, got.WrappedPrivateKey)
	}
	if !reflect.DeepEqual(want.WrappedMasterKey, got.WrappedMasterKey) {
		return fmt.Errorf("wanted wantWrappedMasterKey %v got %v", want.WrappedMasterKey, got.WrappedMasterKey)
	}
	return nil
}

func TestUserCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := setupTestENV(ctx)
	if err != nil {
		t.Fatalf("failed to setup env : %v", err)
	}

	//setup data for test wantUserNoID
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to setup test ecdsa key")
	}
	pk := sk.Public()
	pkPKIXBytes, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		t.Fatalf("failed to setup test ecdsa pubkey encoding")
	}
	ecdsaPK, ok := pk.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("failed to setup generic pubkey as ecdsa pubkey")
	}

	wantEmail := "jon.doe@email.com"
	wantName := "Jon Doe"
	wantWrappedPrivateKey, err := x509.MarshalECPrivateKey(sk)
	wantWrapperMasterKey, err := hex.DecodeString("e57df255399d59a4181c8fb331ca8232")
	if err != nil {
		t.Fatalf("failed to setup mock master key : %v", err)
	}

	wantUserNoID := &domain.User{
		Email:             wantEmail,
		Name:              wantName,
		PublicKey:         *ecdsaPK,
		WrappedPrivateKey: wantWrappedPrivateKey,
		WrappedMasterKey:  wantWrapperMasterKey,
	}

	//create wantUserNoID and check values
	createReq := &UserServiceSchema.UserRequestCreate{
		Email:             wantEmail,
		PublicKey:         pkPKIXBytes,
		WrappedPrivateKey: wantWrappedPrivateKey,
		WrappedMasterKey:  wantWrapperMasterKey,
	}
	gotGRPCUser, err := client.CreateUser(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateUser has unexpected errror :%v", err)
	}

	if err := checkUserNoID(wantUserNoID, gotGRPCUser); err != nil {
		t.Fatalf("unexpected created user content :%v", err)
	}

	//check if the getters return the created user
	gotGRPCUserByEmail, err := client.GetUserByEmail(ctx, &UserServiceSchema.UserRequestEmail{Email: wantEmail})
	if err != nil {
		t.Fatalf("unxepected error fetching wantUserNoID by email : %v", err)
	}
	if !reflect.DeepEqual(gotGRPCUser, gotGRPCUserByEmail) {
		t.Fatalf("user returned by create does not match user returend by get email!")
	}

	gotGRPCUserByID, err := client.GetUserById(ctx, &UserServiceSchema.UserRequestId{Id: gotGRPCUser.Id})
	if err != nil {
		t.Fatalf("unxepected error fetching wantUserNoID by email : %v", err)
	}
	if !reflect.DeepEqual(gotGRPCUser, gotGRPCUserByID) {
		t.Fatalf("user returned by create does not match user returend by get id!")
	}

	//check if deleting works
	_, err = client.DeleteUserById(ctx, &UserServiceSchema.UserRequestId{Id: gotGRPCUser.Id})
	if err != nil {
		t.Fatalf("failed to delete user by id :%v ", err)
	}
	//check that getting the deleted user returns error
	gotGRPCUserByID, err = client.GetUserById(ctx, &UserServiceSchema.UserRequestId{Id: gotGRPCUser.Id})
	if err == nil {
		t.Fatalf("expected error getting deleted user, got none")
	}

	//create new user for checking delete by eamil
	gotGRPCUser, err = client.CreateUser(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateUser has unexpected errror :%v", err)
	}
	//check if deleting works
	_, err = client.DeleteUserByEmail(ctx, &UserServiceSchema.UserRequestEmail{Email: wantEmail})
	if err != nil {
		t.Fatalf("failed to delete user by id :%v ", err)
	}
	//check that getting the deleted user returns error
	gotGRPCUserByID, err = client.GetUserById(ctx, &UserServiceSchema.UserRequestId{Id: gotGRPCUser.Id})
	if err == nil {
		t.Fatalf("expected error getting deleted user, got none")
	}

}
