package userRepository

import (
	"UserService/domain"
	"context"
	"fmt"
	"gorm.io/gorm"
)

type DefaultRepo struct {
	DB *gorm.DB
}

func (d DefaultRepo) GetByPk(ctx context.Context, PKIXPublicKey []byte) (*domain.User, error) {
	dbUser := &UserDTODB{}

	if err := d.DB.WithContext(ctx).Where(" public_key = ?", PKIXPublicKey).First(dbUser).Error; err != nil {
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

func (d DefaultRepo) DeleteByEmail(ctx context.Context, email string) error {
	if err := d.DB.WithContext(ctx).Where("email = ?", email).Delete(&UserDTODB{}).Error; err != nil {
		return err
	}
	return nil
}
