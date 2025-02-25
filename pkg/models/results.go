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
