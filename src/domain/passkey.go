package domain

import (
	"time"
)

type Challenge struct {
	ID        string    `gorm:"primaryKey"`
	Domain    string    `gorm:"not null"`
	Challenge string    `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
