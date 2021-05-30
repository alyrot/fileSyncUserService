package services

import (
	"UserService/adapters/userRepository"
	"UserService/domain"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
)

func NewUserService(userRepo userRepository.Repo) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

type UserService struct {
	UnimplementedUserServiceServer
	userRepo userRepository.Repo
}

func userToDTOGRPC(u *domain.User) (*User, error) {
	pkPKIX, err := x509.MarshalPKIXPublicKey(&u.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert .PublicKey field : %v", err)
	}
	grpcUser := &User{
		Id:                uint64(u.ID),
		CreatedAtUnix:     u.CreatedAt.Unix(),
		UpdatedAtUnix:     u.UpdatedAt.Unix(),
		Email:             u.Email,
		PublicKey:         pkPKIX,
		WrappedPrivateKey: u.WrappedPrivateKey,
		WrappedMasterKey:  u.WrappedMasterKey,
	}
	return grpcUser, nil
}

func (us *UserService) GetUserById(ctx context.Context, userRequest *UserRequestId) (*User, error) {
	domainUser, err := us.userRepo.GetById(ctx, uint(userRequest.Id))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user :%v", err)
	}
	grpcUser, err := userToDTOGRPC(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to convert domain user to dto : %v", err)
	}
	return grpcUser, nil

}
func (us *UserService) GetUserByEmail(ctx context.Context, userRequest *UserRequestEmail) (*User, error) {
	domainUser, err := us.userRepo.GetByEmail(ctx, userRequest.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user :%v", err)
	}
	grpcUser, err := userToDTOGRPC(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to convert domain user to dto : %v", err)
	}
	return grpcUser, nil
}

func (us *UserService) CreateUser(ctx context.Context, req *UserRequestCreate) (*User, error) {

	genericPK, err := x509.ParsePKIXPublicKey(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key : %v", err)
	}
	pk, ok := genericPK.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key has to be ecdsa")
	}

	domainUser := &domain.User{
		Email:             req.Email,
		PublicKey:         *pk,
		WrappedPrivateKey: req.WrappedPrivateKey,
		WrappedMasterKey:  req.WrappedMasterKey,
	}
	domainUser, err = us.userRepo.Create(ctx, domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to crate user :%v", err)
	}
	grpcUser, err := userToDTOGRPC(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize user :%v", err)
	}
	return grpcUser, nil
}

func (us *UserService) DeleteUserById(ctx context.Context, req *UserRequestId) (*Empty, error) {
	if err := us.userRepo.DeleteById(ctx, uint(req.Id)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}
func (us *UserService) DeleteUserByEmail(ctx context.Context, req *UserRequestEmail) (*Empty, error) {
	if err := us.userRepo.DeleteByEmail(ctx, req.Email); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

func (us *UserService) GetUserPkById(ctx context.Context, userRequest *UserRequestId) (*UserPk, error) {
	domainUser, err := us.userRepo.GetById(ctx, uint(userRequest.Id))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user :%v", err)
	}
	grpcUser, err := userToDTOGRPC(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to convert domain user to dto : %v", err)
	}
	userPK := &UserPk{
		Id:        grpcUser.Id,
		Email:     grpcUser.Email,
		PublicKey: grpcUser.PublicKey,
	}
	return userPK, nil
}
func (us *UserService) GetUserPkByEmail(ctx context.Context, userRequest *UserRequestEmail) (*UserPk, error) {
	domainUser, err := us.userRepo.GetByEmail(ctx, userRequest.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user :%v", err)
	}
	grpcUser, err := userToDTOGRPC(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to convert domain user to dto : %v", err)
	}
	userPK := &UserPk{
		Id:        grpcUser.Id,
		Email:     grpcUser.Email,
		PublicKey: grpcUser.PublicKey,
	}
	return userPK, nil
}
