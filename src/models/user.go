package models

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type User struct {
	ID              uint   `json:"id" validate:"required"`
	Name            string `json:"name" validate:"required"`
	Email           string `json:"email" validate:"required"`
	Description     string `json:"description"`
	UID             string `json:"uid" validate:"required"`
	ParentId        uint   `json:"parent_id"`
	EmailVerifiedAt string `json:"email_verified_at"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	DeletedAt       string `json:"deleted_at"`
}

func GetUserById(db *gorm.DB, id int) (*User, error) {
	var user User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid user UID: %d", id)
		}
		return nil, err
	}
	return &user, nil
}
