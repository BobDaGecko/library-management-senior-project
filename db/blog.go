package db

import (
	"time"
)

var _ = Migrate(Blog{})

type Blog struct {
	BaseModel
	Title  string
	Body   string
	Date   time.Time
	Tags   SqlStringList `gorm:"type:text"`
	User   User
	UserID SqlUUID `gorm:"type:text"`
}
