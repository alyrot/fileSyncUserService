package domain

import (
	"crypto"
	"fmt"
	"time"
)

type User struct {
	Email             string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Name              string
	PublicKey         crypto.PublicKey
	WrappedPrivateKey []byte
	WrappedMasterKey  []byte
}

func (u User) String() string {
	return fmt.Sprintf("User{CreatedAt %v, Email: %v, Name: %v, PublicKey: %v, WrappedPrivateKey: %v, WrappedMasterKey: %v}",
		u.CreatedAt, u.Email, u.Name, u.PublicKey, u.WrappedMasterKey, u.WrappedMasterKey)
}
