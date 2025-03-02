package models

import (
	"errors"
	"strings"

	"github.com/geobattles/geobattles-backend/pkg/auth"
	"github.com/geobattles/geobattles-backend/pkg/logic"
	"gorm.io/gorm"
)

type BaseUser struct {
	ID          string `gorm:"primary_key;type:varchar(6);not null;unique"`
	DisplayName string `gorm:"type:varchar(255);not null"`
	IsGuest     bool   `gorm:"not null;default:false"`
}

type User struct {
	BaseUser
	UserName string `gorm:"type:varchar(255);unique"`
	Password string `gorm:"type:varchar(255)"`
}

type Guest struct {
	BaseUser
}

// set table name for both User and Guest
func (BaseUser) TableName() string {
	return "users"
}

// BeforeCreate GORM hook for baseUser
func (baseUser *BaseUser) BeforeCreate(tx *gorm.DB) (err error) {
	// TODO: check for existing duplicate ID
	baseUser.ID = logic.GenerateRndID(6)

	return nil
}

// BeforeCreate GORM hook to handle pre-processing before saving a user to the database
func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	if err := user.BaseUser.BeforeCreate(tx); err != nil {
		return err
	}

	if user.UserName == "" {
		return errors.New("userName cannot be empty")
	}

	// Set DisplayName to Name if DisplayName is not provided
	if user.DisplayName == "" {
		user.DisplayName = user.UserName
	}

	user.UserName = strings.ToLower(user.UserName)

	if user.Password == "" {
		return errors.New("password cannot be empty")
	}
	// Hash the password before creating the record
	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		return err
	}

	user.Password = hashedPassword

	user.IsGuest = false

	return nil
}

// BeforeCreate GORM hook to handle pre-processing before saving a guest to the database
func (guest *Guest) BeforeCreate(tx *gorm.DB) (err error) {
	if err := guest.BaseUser.BeforeCreate(tx); err != nil {
		return err
	}

	if guest.DisplayName == "" {
		return errors.New("displayName cannot be empty")
	}

	guest.IsGuest = true

	return nil
}

// BeforeUpdate GORM hook to handle pre-processing before updating a user in the database
func (user *User) BeforeUpdate(tx *gorm.DB) (err error) {
	// Hash the password before updating the record
	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		return err
	}

	user.Password = hashedPassword

	return nil
}
