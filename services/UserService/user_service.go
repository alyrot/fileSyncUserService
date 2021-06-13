package UserService

import (
	"UserService/adapters/userRepository"
	"UserService/domain"
	"UserService/protobufs/UserServiceSchema"
	"context"
	"crypto/x509"
	"fmt"
)

func NewUserService(userRepo userRepository.UserRepo) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

type UserService struct {
	UserServiceSchema.UnimplementedUserServiceServer
	userRepo userRepository.UserRepo
}

func userToDTOGRPC(u *domain.User) (*UserServiceSchema.User, error) {
	pkPKIX, err := x509.MarshalPKIXPublicKey(u.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert .PublicKey field : %v", err)
	}
	grpcUser := &UserServiceSchema.User{
		CreatedAtUnix:     u.CreatedAt.Unix(),
		UpdatedAtUnix:     u.UpdatedAt.Unix(),
		Email:             u.Email,
		PublicKey:         pkPKIX,
		WrappedPrivateKey: u.WrappedPrivateKey,
		WrappedMasterKey:  u.WrappedMasterKey,
	}
	return grpcUser, nil
}

func (us *UserService) GetUserByPk(ctx context.Context, userRequest *UserServiceSchema.UserRequestPk) (*UserServiceSchema.User, error) {
	domainUser, err := us.userRepo.GetByPk(ctx, userRequest.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user :%v", err)
	}
	grpcUser, err := userToDTOGRPC(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to convert domain user to dto : %v", err)
	}
	return grpcUser, nil

}
func (us *UserService) GetUserByEmail(ctx context.Context, userRequest *UserServiceSchema.UserRequestEmail) (*UserServiceSchema.User, error) {
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

func (us *UserService) CreateUser(ctx context.Context, req *UserServiceSchema.UserRequestCreate) (*UserServiceSchema.User, error) {
	genericPK, err := x509.ParsePKIXPublicKey(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key : %v", err)
	}

	domainUser := &domain.User{
		Email:             req.Email,
		PublicKey:         genericPK,
		WrappedPrivateKey: req.WrappedPrivateKey,
		WrappedMasterKey:  req.WrappedMasterKey,
	}
	domainUser, err = us.userRepo.Create(ctx, domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user :%v", err)
	}
	grpcUser, err := userToDTOGRPC(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize user :%v", err)
	}
	return grpcUser, nil
}

func (us *UserService) DeleteUserByEmail(ctx context.Context, req *UserServiceSchema.UserRequestEmail) (*UserServiceSchema.Empty, error) {
	if err := us.userRepo.DeleteByEmail(ctx, req.Email); err != nil {
		return nil, err
	}
	return &UserServiceSchema.Empty{}, nil
}

func (us *UserService) GetUserPkByEmail(ctx context.Context, userRequest *UserServiceSchema.UserRequestEmail) (*UserServiceSchema.UserPk, error) {
	domainUser, err := us.userRepo.GetByEmail(ctx, userRequest.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user :%v", err)
	}
	grpcUser, err := userToDTOGRPC(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to convert domain user to dto : %v", err)
	}
	userPK := &UserServiceSchema.UserPk{
		Email:     grpcUser.Email,
		PublicKey: grpcUser.PublicKey,
	}
	return userPK, nil
}
