package models

import "gorm.io/gorm"

type Message struct {
	gorm.Model
	SessionID   string
	ID          uint  `gorm:"PrimaryKey;autoIncrement"`
	TimeCreated int64 `gorm:"autoCreateTime"`
	TimeUpdated int64 `gorm:"autoUpdateTime:nano"`
	Data        string
}
