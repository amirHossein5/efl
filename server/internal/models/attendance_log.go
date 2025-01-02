package models

import (
	"gorm.io/gorm"
)

type AttendanceLog struct {
	gorm.Model
	UserID uint64
}
