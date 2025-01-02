package models

import (
	"gorm.io/gorm"
)

type EnrolledFace struct {
	gorm.Model
	UserID uint64
	Path   string
}
