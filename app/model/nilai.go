package model

import (
	"time"

	"gorm.io/gorm"
)

type Nilai struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	MahasiswaID   uint           `gorm:"not null" json:"mahasiswa_id"`
	MataKuliahID  uint           `gorm:"not null" json:"mata_kuliah_id"`
	Nilai         float64        `gorm:"type:decimal(5,2);not null" json:"nilai"`
	Semester      int            `gorm:"not null" json:"semester"`
	Mahasiswa     Mahasiswa      `gorm:"foreignKey:MahasiswaID" json:"mahasiswa,omitempty"`
	MataKuliah    MataKuliah     `gorm:"foreignKey:MataKuliahID" json:"mata_kuliah,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Nilai) TableName() string {
	return "nilai"
}
