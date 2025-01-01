package dbconnection

import (
	"log"

	"gorm.io/gorm"
)

var Conn *gorm.DB

func Open(dialector gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
	log.Println("initializing database connection...")

	db, err := gorm.Open(dialector, config)

	if err == nil {
		Conn = db
	}

	return db, err
}
