package domain

import (
	"crypto/ecdsa"
	"fmt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email             string
	Name              string
	PublicKey         ecdsa.PublicKey
	WrappedPrivateKey []byte
	WrappedMasterKey  []byte
}

func (u User) String() string {
	return fmt.Sprintf("User{ID: %v, CreatedAt %v, Email: %v, Name: %v, PublicKey: %v, WrappedPrivateKey: %v, WrappedMasterKey: %v}",
		u.ID, u.CreatedAt, u.Email, u.Name, u.PublicKey, u.WrappedMasterKey, u.WrappedMasterKey)
}
