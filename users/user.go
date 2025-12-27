package users

import (
	"time"

	"gorm.io/gorm"
)

type Users struct {
	Id        uint64         `gorm:"primaryKey;column:id" json:"id"`
	NickName  string         `gorm:"column:nick_name;not null;size:100" json:"nick_name"`
	Email     *string        `gorm:"column:email;size:255" json:"email"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
	Phone     string         `gorm:"column:phone;unique;not null;size:20" json:"phone"`
	Rate      uint64         `gorm:"column:rate; type: numeric" json:"rate"`
}

func (Users) TableName() string {
	return "users"
}
