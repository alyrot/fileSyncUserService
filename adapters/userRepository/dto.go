package userRepository

import (
	"UserService/domain"
	"crypto/x509"
	"fmt"
	"time"
)

func userToDTODB(u *domain.User) (*UserDTODB, error) {
	pkPKIX, err := x509.MarshalPKIXPublicKey(u.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert .PublicKey field : %v", err)
	}
	return &UserDTODB{
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
		Email:             u.Email,
		Name:              u.Name,
		PublicKeyPKIX:     pkPKIX,
		WrappedPrivateKey: u.WrappedPrivateKey,
		WrappedMasterKey:  u.WrappedMasterKey,
	}, nil
}

type UserDTODB struct {
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Email             string `gorm:"primaryKey"`
	Name              string `gorm:"not null"`
	PublicKeyPKIX     []byte `gorm:"not null"`
	WrappedPrivateKey []byte `gorm:"not null"`
	WrappedMasterKey  []byte `gorm:"not null"`
}

func (u *UserDTODB) toUser() (*domain.User, error) {
	genericPubKey, err := x509.ParsePKIXPublicKey(u.PublicKeyPKIX)
	if err != nil {
		return nil, fmt.Errorf(".PublicKey is no valid x509.PKIX pubkey")
	}
	return &domain.User{
		Email:             u.Email,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
		Name:              u.Name,
		PublicKey:         genericPubKey,
		WrappedPrivateKey: u.WrappedPrivateKey,
		WrappedMasterKey:  u.WrappedMasterKey,
	}, err

}
