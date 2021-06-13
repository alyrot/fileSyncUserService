package userRepository

import (
	"UserService/domain"
	"context"
	"errors"
)

var ErrNotFound = errors.New("entry not found")
var ErrAlreadyExists = errors.New("entry already exists")

type UserRepo interface {
	GetByPk(ctx context.Context, PKIXPublicKey []byte) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	DeleteByEmail(ctx context.Context, email string) error
}
