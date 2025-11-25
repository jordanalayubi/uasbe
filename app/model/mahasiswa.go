package model

import (
	"time"

	"gorm.io/gorm"
)

type Mahasiswa struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	NIM       string         `gorm:"unique;not null" json:"nim"`
	Nama      string         `gorm:"not null" json:"nama"`
	Email     string         `gorm:"unique;not null" json:"email"`
	Jurusan   string         `gorm:"not null" json:"jurusan"`
	Angkatan  int            `gorm:"not null" json:"angkatan"`
	UserID    uint           `gorm:"not null" json:"user_id"`
	User      User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Mahasiswa) TableName() string {
	return "mahasiswa"
}
