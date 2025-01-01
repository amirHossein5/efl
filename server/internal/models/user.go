package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	CreatedAt      time.Time       `gorm:"default:CURRENT_TIMESTAMP"`
	AttendanceLogs []AttendanceLog `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	EnrolledFaces  []EnrolledFace  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
