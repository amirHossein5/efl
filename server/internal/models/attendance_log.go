package models

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	ATTENDANCE_LOG_TYPE_ENTERED = "attendance_log_type_entered"
	ATTENDANCE_LOG_TYPE_EXITED  = "attendance_log_type_exited"
)

type AttendanceLog struct {
	gorm.Model
	UserID uint64
	Type   string
}

func (attendanceLog *AttendanceLog) BeforeCreate(tx *gorm.DB) error {
	if attendanceLog.Type == "" {
		var todayUserLogsCount int64

		err := tx.Model(&AttendanceLog{}).Where("user_id = ?", attendanceLog.UserID).Where("DATE(created_at) = ?", time.Now().Format("2006-01-02")).Count(&todayUserLogsCount).Error
		if err != nil {
			return fmt.Errorf("AttendanceLog,BeforeCreate: %w", err)
		}

		if todayUserLogsCount%2 == 0 {
			attendanceLog.Type = ATTENDANCE_LOG_TYPE_ENTERED
		} else {
			attendanceLog.Type = ATTENDANCE_LOG_TYPE_EXITED
		}
	}

	return nil
}
