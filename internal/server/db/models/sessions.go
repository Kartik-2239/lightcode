package models

import "gorm.io/gorm"

//sessions

type Session struct {
	gorm.Model
	ID          string `gorm:"primaryKey"`
	Directory   string
	Title       string
	TimeCreated int64 `gorm:"autoCreateTime"`
	TimeUpdated int64 `gorm:"autoUpdateTime:nano"`
}
