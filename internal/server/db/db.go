package db

import (
	"github.com/Kartik-2239/lightcode/internal/server/db/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("/Users/kartikkannan/Desktop/lightcode/lightcode.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&models.Session{},
		&models.Message{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
