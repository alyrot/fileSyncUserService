package main

import (
	"UserService/adapters/userRepository"
	"UserService/domain"
	"UserService/protobufs/UserServiceSchema"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"log"
	"net"
	"reflect"
	"testing"
)

type dbImpl string

func (d dbImpl) String() string {
	return string(d)
}

const gormDbImpl = dbImpl("gorm")
const dynamoDbImpl = dbImpl("dynamo")

//setupTestENV creates a client connected to an in memory service using an inmemory db
func setupTestENV(ctx context.Context, backend dbImpl) (UserServiceSchema.UserServiceClient, error) {
	var userRepo userRepository.UserRepo
	switch backend {
	case gormDbImpl:
		//this will create an in memory sqlite db
		dsn := "file::memory:?cache=shared"
		db, err := SetupGormDB(dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to create %v backend : %v", backend, err)
		}
		userRepo = &userRepository.DefaultRepo{DB: db}
	case dynamoDbImpl:
		log.Printf("Testing dynamo db backend, make sure docker container is running!")
		sess := session.Must(session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentials("test-id", "test-secret", "test-token"),
			Region:      aws.String("us-west-2"),
		}))
		var err error
		userRepo, err = userRepository.NewAwsLocalDynamoUserRepo(sess)
		if err != nil {
			return nil, fmt.Errorf("failed to create %v backend : %v", backend, err)
		}
	default:
		return nil, fmt.Errorf("unknown db implementation %v", backend)

	}

	//create server
	bufferSize := 1024 * 1024
	lis := bufconn.Listen(bufferSize)
	server := SetupGRPCServer(userRepo)
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
	return client, nil
}

func checkUserNoID(want *domain.User, got *UserServiceSchema.User) error {
	if want.Email != got.Email {
		return fmt.Errorf("want email %v got %v", want.Email, got.Email)
	}

	wantPkBytes, err := x509.MarshalPKIXPublicKey(want.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse wanted public key to bytes : %v", err)
	}
	if !reflect.DeepEqual(wantPkBytes, got.PublicKey) {
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

func testUserCreationWithBackend(ctx context.Context, t *testing.T, client UserServiceSchema.UserServiceClient) {
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
		PublicKey:         ecdsaPK,
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

	//check if deleting works
	_, err = client.DeleteUserByEmail(ctx, &UserServiceSchema.UserRequestEmail{Email: wantEmail})
	if err != nil {
		t.Fatalf("failed to delete user by id :%v ", err)
	}
	//check that getting the deleted user returns error
	gotGRPCUserByEmail, err = client.GetUserByEmail(ctx, &UserServiceSchema.UserRequestEmail{Email: gotGRPCUser.Email})
	if err == nil {
		t.Fatalf("expected error getting deleted user, got none")
	}
}

func TestUserCreation(t *testing.T) {
	backends := []dbImpl{dynamoDbImpl, gormDbImpl}
	for _, v := range backends {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client, err := setupTestENV(ctx, v)
			if err != nil {
				t.Fatalf("failed to setup env : %v", err)
			}

			testUserCreationWithBackend(ctx, t, client)

		})
	}

}
