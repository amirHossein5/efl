package models

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/amirhossein5/efl/server/internal/dbconnection"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	AttendanceLogs []AttendanceLog `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	EnrolledFaces  []EnrolledFace  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (user *User) LogAttendance() error {
	can, err := user.CanLogAttendance()
	if can && err == nil {
		attendanceLog := AttendanceLog{UserID: uint64(user.ID)}
		err := dbconnection.Conn.Create(&attendanceLog).Error
		return err
	}

	if err != nil {
		return err
	}
	return fmt.Errorf("CanLogAttendance returned false")
}

func (user *User) CanLogAttendance() (bool, error) {
	var attendanceLog AttendanceLog
	err := dbconnection.Conn.Where("user_id = ?", user.ID).Last(&attendanceLog).Error

	if err != nil {
		log.Println(errors.Is(err, gorm.ErrRecordNotFound))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}

	return time.Now().Add(-15 * time.Second).After(attendanceLog.CreatedAt), nil
}
