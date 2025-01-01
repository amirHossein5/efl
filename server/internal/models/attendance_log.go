package models

import (
	"time"

	"gorm.io/gorm"
)

type AttendanceLog struct {
	gorm.Model
	UserID    uint64
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}
