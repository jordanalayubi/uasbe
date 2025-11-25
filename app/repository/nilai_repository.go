package repository

import (
	"UASBE/app/model"

	"gorm.io/gorm"
)

type NilaiRepository interface {
	Create(nilai *model.Nilai) error
	FindAll() ([]model.Nilai, error)
	FindByID(id uint) (*model.Nilai, error)
	FindByMahasiswaID(mahasiswaID uint) ([]model.Nilai, error)
	Update(nilai *model.Nilai) error
	Delete(id uint) error
}

type nilaiRepository struct {
	db *gorm.DB
}

func NewNilaiRepository(db *gorm.DB) NilaiRepository {
	return &nilaiRepository{db: db}
}

func (r *nilaiRepository) Create(nilai *model.Nilai) error {
	return r.db.Create(nilai).Error
}

func (r *nilaiRepository) FindAll() ([]model.Nilai, error) {
	var nilai []model.Nilai
	err := r.db.Preload("Mahasiswa").Preload("MataKuliah").Find(&nilai).Error
	return nilai, err
}

func (r *nilaiRepository) FindByID(id uint) (*model.Nilai, error) {
	var nilai model.Nilai
	err := r.db.Preload("Mahasiswa").Preload("MataKuliah").First(&nilai, id).Error
	return &nilai, err
}

func (r *nilaiRepository) FindByMahasiswaID(mahasiswaID uint) ([]model.Nilai, error) {
	var nilai []model.Nilai
	err := r.db.Preload("Mahasiswa").Preload("MataKuliah").Where("mahasiswa_id = ?", mahasiswaID).Find(&nilai).Error
	return nilai, err
}

func (r *nilaiRepository) Update(nilai *model.Nilai) error {
	return r.db.Save(nilai).Error
}

func (r *nilaiRepository) Delete(id uint) error {
	return r.db.Delete(&model.Nilai{}, id).Error
}
