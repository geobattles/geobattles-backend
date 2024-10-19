package models

import "errors"

// user model
type User struct {
	ID       uint   `gorm:"primary_key;auto_increment"`
	Name     string `gorm:"type:varchar(255);not null;unique"`
	Password string `gorm:"not null"`
}

// validate that user fields are not empty
func (u *User) ValidateUser() error {
	if u.Name == "" {
		return errors.New("name cannot be empty")
	}
	if u.Password == "" {
		return errors.New("nassword cannot be empty")
	}
	return nil
}
