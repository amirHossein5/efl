package models

import (
	"time"

	"gorm.io/gorm"
)

type EnrolledFace struct {
	gorm.Model
	UserID    uint64
	Path      string
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
