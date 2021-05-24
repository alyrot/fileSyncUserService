package userRepository

import (
	"UserService/domain"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"gorm.io/gorm"
)

func userToDTODB(u *domain.User) (*UserDTODB, error) {
	pkPKIX, err := x509.MarshalPKIXPublicKey(&u.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert .PublicKey field : %v", err)
	}
	return &UserDTODB{
		Model:             u.Model,
		Email:             u.Email,
		Name:              u.Name,
		PublicKeyPKIX:     pkPKIX,
		WrappedPrivateKey: u.WrappedPrivateKey,
		WrappedMasterKey:  u.WrappedMasterKey,
	}, nil
}

type UserDTODB struct {
	gorm.Model
	Email             string `gorm:not null,uniqueIndex`
	Name              string `gorm:not null`
	PublicKeyPKIX     []byte `gorm:not null`
	WrappedPrivateKey []byte `gorm:not null`
	WrappedMasterKey  []byte `gorm:not null`
}

func (u *UserDTODB) toUser() (*domain.User, error) {
	genericPubKey, err := x509.ParsePKIXPublicKey(u.PublicKeyPKIX)
	if err != nil {
		return nil, fmt.Errorf(".PublicKey is no valid x509.PKIX pubkey")
	}
	pk, ok := genericPubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf(".PublicKey is valid x509.PKIX but is no golang ecdsa.PublicKey")
	}
	return &domain.User{
		Model:             u.Model,
		Email:             u.Email,
		Name:              u.Name,
		PublicKey:         *pk,
		WrappedPrivateKey: u.WrappedPrivateKey,
		WrappedMasterKey:  u.WrappedMasterKey,
	}, err

}

type Repo interface {
	GetById(ctx context.Context, id uint) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	DeleteById(ctx context.Context, id uint) error
	DeleteByEmail(ctx context.Context, email string) error
}

type DefaultRepo struct {
	DB *gorm.DB
}

func (d DefaultRepo) GetById(ctx context.Context, id uint) (*domain.User, error) {
	dbUser := &UserDTODB{}
	if err := d.DB.WithContext(ctx).Find(dbUser, id).Error; err != nil {
		return nil, err
	}
	user, err := dbUser.toUser()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to user :%v", err)
	}
	return user, nil
}

func (d DefaultRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	dbUser := &UserDTODB{}
	if err := d.DB.WithContext(ctx).Where("email = ?", email).First(dbUser).Error; err != nil {
		return nil, err
	}
	user, err := dbUser.toUser()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to user :%v", err)
	}
	return user, nil
}

func (d DefaultRepo) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	dbUser, err := userToDTODB(u)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize user for DB : %v", err)
	}
	if err := d.DB.WithContext(ctx).Create(dbUser).Error; err != nil {
		return nil, fmt.Errorf("failed to insert user : %v", err)
	}
	user, err := dbUser.toUser()
	if err != nil {
		return nil, fmt.Errorf("failed to update user id after create :%v", err)
	}
	return user, nil
}

func (d DefaultRepo) DeleteById(ctx context.Context, id uint) error {
	if err := d.DB.WithContext(ctx).Delete(&UserDTODB{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (d DefaultRepo) DeleteByEmail(ctx context.Context, email string) error {
	if err := d.DB.WithContext(ctx).Where("email = ?", email).Delete(&UserDTODB{}).Error; err != nil {
		return err
	}
	return nil
}
