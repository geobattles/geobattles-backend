package models

import (
	"time"

	"github.com/slinarji/go-geo-server/pkg/logic"
)

type Game struct {
	ID        uint      `gorm:"primary_key" json:"-"`
	Timestamp time.Time `gorm:"timestamp"`
	Rounds    []Round   `gorm:"foreignKey:GameID"`
}

type Round struct {
	ID     uint         `gorm:"primary_key" json:"-"`
	GameID uint         `json:"-"`
	Loc    logic.Coords `gorm:"embedded;embeddedPrefix:loc_"`
	// Country string       `gorm:"country"`
	Results []Result `gorm:"foreignKey:RoundID"`
}

type Result struct {
	ID          uint         `gorm:"primaryKey" json:"-"`
	RoundID     uint         `json:"-"`
	UserID      string       `gorm:"type:varchar(6);not null" json:"-"`
	Loc         logic.Coords `gorm:"embedded;embeddedPrefix:loc_" json:"location"`
	Dist        float64      `gorm:"distance" json:"distance"`
	BaseScore   int          `gorm:"-" json:"baseScr"`
	PlaceScore  int          `gorm:"-" json:"placeScr,omitempty"`
	DoubleScore int          `gorm:"-" json:"dblScr,omitempty"`
	DuelScore   int          `gorm:"-" json:"duelScr,omitempty"`
	Total       int          `gorm:"score" json:"total,omitempty"`
	Attempt     int          `gorm:"-" json:"attempt"`
	Lives       int          `gorm:"-" json:"lives"`
	CC          string       `gorm:"-" json:"cc,omitempty"`
	Time        int          `gorm:"-" json:"time,omitempty"`
}

// // set table name for both User and Guest
// func (BaseUser) TableName() string {
// 	return "users"
// }

// // BeforeCreate GORM hook for baseUser
// func (baseUser *BaseUser) BeforeCreate(tx *gorm.DB) (err error) {
// 	// TODO: check for existing duplicate ID
// 	baseUser.ID = logic.GenerateRndID(6)

// 	return nil
// }

// // BeforeCreate GORM hook to handle pre-processing before saving a user to the database
// func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
// 	if err := user.BaseUser.BeforeCreate(tx); err != nil {
// 		return err
// 	}

// 	if user.UserName == "" {
// 		return errors.New("userName cannot be empty")
// 	}

// 	// Set DisplayName to Name if DisplayName is not provided
// 	if user.DisplayName == "" {
// 		user.DisplayName = user.UserName
// 	}

// 	user.UserName = strings.ToLower(user.UserName)

// 	if user.Password == "" {
// 		return errors.New("password cannot be empty")
// 	}
// 	// Hash the password before creating the record
// 	hashedPassword, err := auth.HashPassword(user.Password)
// 	if err != nil {
// 		return err
// 	}

// 	user.Password = hashedPassword

// 	user.IsGuest = false

// 	return nil
// }

// // BeforeCreate GORM hook to handle pre-processing before saving a guest to the database
// func (guest *Guest) BeforeCreate(tx *gorm.DB) (err error) {
// 	if err := guest.BaseUser.BeforeCreate(tx); err != nil {
// 		return err
// 	}

// 	if guest.DisplayName == "" {
// 		return errors.New("displayName cannot be empty")
// 	}

// 	guest.IsGuest = true

// 	return nil
// }
