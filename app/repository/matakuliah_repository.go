package repository

import (
	"UASBE/app/model"

	"gorm.io/gorm"
)

type MataKuliahRepository interface {
	Create(matakuliah *model.MataKuliah) error
	FindAll() ([]model.MataKuliah, error)
	FindByID(id uint) (*model.MataKuliah, error)
	FindByKodeMK(kodeMK string) (*model.MataKuliah, error)
	Update(matakuliah *model.MataKuliah) error
	Delete(id uint) error
}

type mataKuliahRepository struct {
	db *gorm.DB
}

func NewMataKuliahRepository(db *gorm.DB) MataKuliahRepository {
	return &mataKuliahRepository{db: db}
}

func (r *mataKuliahRepository) Create(matakuliah *model.MataKuliah) error {
	return r.db.Create(matakuliah).Error
}

func (r *mataKuliahRepository) FindAll() ([]model.MataKuliah, error) {
	var matakuliah []model.MataKuliah
	err := r.db.Find(&matakuliah).Error
	return matakuliah, err
}

func (r *mataKuliahRepository) FindByID(id uint) (*model.MataKuliah, error) {
	var matakuliah model.MataKuliah
	err := r.db.First(&matakuliah, id).Error
	return &matakuliah, err
}

func (r *mataKuliahRepository) FindByKodeMK(kodeMK string) (*model.MataKuliah, error) {
	var matakuliah model.MataKuliah
	err := r.db.Where("kode_mk = ?", kodeMK).First(&matakuliah).Error
	return &matakuliah, err
}

func (r *mataKuliahRepository) Update(matakuliah *model.MataKuliah) error {
	return r.db.Save(matakuliah).Error
}

func (r *mataKuliahRepository) Delete(id uint) error {
	return r.db.Delete(&model.MataKuliah{}, id).Error
}
