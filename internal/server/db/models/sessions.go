package models

import (
	"gorm.io/gorm"
)

type Session struct {
	gorm.Model
	ID          string `gorm:"primaryKey"`
	Directory   string
	ToDoList    string `gorm:"type:longtext"`
	Title       string
	TimeCreated int64 `gorm:"autoCreateTime"`
	TimeUpdated int64 `gorm:"autoUpdateTime:nano"`
}
