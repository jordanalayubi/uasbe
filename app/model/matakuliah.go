package model

import (
	"time"

	"gorm.io/gorm"
)

type MataKuliah struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	KodeMK    string         `gorm:"unique;not null" json:"kode_mk"`
	NamaMK    string         `gorm:"not null" json:"nama_mk"`
	SKS       int            `gorm:"not null" json:"sks"`
	Semester  int            `gorm:"not null" json:"semester"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MataKuliah) TableName() string {
	return "mata_kuliah"
}
