package models

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type User struct {
	ID          uint   `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Email       string `json:"email" validate:"required"`
	Description string `json:"description"`
	UID         string `json:"uid" validate:"required"`
	ParentId    uint   `json:"parent_id"`
}

func GetUserByUID(db *gorm.DB, uid string) (*User, error) {
	var user User
	if err := db.Where("uid = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid user UID: %s", uid)
		}
		return nil, err
	}
	return &user, nil
}
