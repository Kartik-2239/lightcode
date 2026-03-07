package models

import "gorm.io/gorm"

type Message struct {
	gorm.Model
	SessionID   string
	ID          string `gorm:"PrimaryKey"`
	TimeCreated int64  `gorm:"autoCreateTime"`
	TimeUpdated int64  `gorm:"autoUpdateTime:nano"`
	Data        string
}
