package models

import "time"

type User struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"size:255;not null"`
	Email       string    `gorm:"size:320;not null;uniqueIndex"`
	PhoneNumber string    `gorm:"size:32;not null"`
	Addresses   []Address `gorm:"foreignKey:UserID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (User) TableName() string {
	return "users"
}

type Address struct {
	ID        int64  `gorm:"primaryKey"`
	UserID    string `gorm:"type:uuid;index;not null"`
	Street    string `gorm:"size:255;not null"`
	City      string `gorm:"size:120;not null"`
	State     string `gorm:"size:120;not null"`
	ZipCode   string `gorm:"size:20;not null"`
	Country   string `gorm:"size:120;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Address) TableName() string {
	return "addresses"
}
