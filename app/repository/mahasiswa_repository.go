package repository

import (
	"UASBE/app/model"

	"gorm.io/gorm"
)

type MahasiswaRepository interface {
	Create(mahasiswa *model.Mahasiswa) error
	FindAll() ([]model.Mahasiswa, error)
	FindByID(id uint) (*model.Mahasiswa, error)
	FindByNIM(nim string) (*model.Mahasiswa, error)
	Update(mahasiswa *model.Mahasiswa) error
	Delete(id uint) error
}

type mahasiswaRepository struct {
	db *gorm.DB
}

func NewMahasiswaRepository(db *gorm.DB) MahasiswaRepository {
	return &mahasiswaRepository{db: db}
}

func (r *mahasiswaRepository) Create(mahasiswa *model.Mahasiswa) error {
	return r.db.Create(mahasiswa).Error
}

func (r *mahasiswaRepository) FindAll() ([]model.Mahasiswa, error) {
	var mahasiswa []model.Mahasiswa
	err := r.db.Preload("User").Find(&mahasiswa).Error
	return mahasiswa, err
}

func (r *mahasiswaRepository) FindByID(id uint) (*model.Mahasiswa, error) {
	var mahasiswa model.Mahasiswa
	err := r.db.Preload("User").First(&mahasiswa, id).Error
	return &mahasiswa, err
}

func (r *mahasiswaRepository) FindByNIM(nim string) (*model.Mahasiswa, error) {
	var mahasiswa model.Mahasiswa
	err := r.db.Preload("User").Where("nim = ?", nim).First(&mahasiswa).Error
	return &mahasiswa, err
}

func (r *mahasiswaRepository) Update(mahasiswa *model.Mahasiswa) error {
	return r.db.Save(mahasiswa).Error
}

func (r *mahasiswaRepository) Delete(id uint) error {
	return r.db.Delete(&model.Mahasiswa{}, id).Error
}
