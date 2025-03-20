package models

type RefreshToken struct {
	ID        string `gorm:"primary_key;type:varchar(36)"`
	UserID    string `gorm:"type:varchar(6);not null;index"`
	Revoked   bool   `gorm:"not null;default:false"`
	ExpiresAt int64  `gorm:"not null"`
	CreatedAt int64  `gorm:"not null"`
}
