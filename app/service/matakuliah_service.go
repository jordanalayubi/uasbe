package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"
	"errors"

	"gorm.io/gorm"
)

type MataKuliahService interface {
	Create(matakuliah *model.MataKuliah) error
	GetAll() ([]model.MataKuliah, error)
	GetByID(id uint) (*model.MataKuliah, error)
	Update(id uint, matakuliah *model.MataKuliah) error
	Delete(id uint) error
}

type mataKuliahService struct {
	matakuliahRepo repository.MataKuliahRepository
}

func NewMataKuliahService(matakuliahRepo repository.MataKuliahRepository) MataKuliahService {
	return &mataKuliahService{matakuliahRepo: matakuliahRepo}
}

func (s *mataKuliahService) Create(matakuliah *model.MataKuliah) error {
	// Check if KodeMK already exists
	_, err := s.matakuliahRepo.FindByKodeMK(matakuliah.KodeMK)
	if err == nil {
		return errors.New("Kode MK already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return s.matakuliahRepo.Create(matakuliah)
}

func (s *mataKuliahService) GetAll() ([]model.MataKuliah, error) {
	return s.matakuliahRepo.FindAll()
}

func (s *mataKuliahService) GetByID(id uint) (*model.MataKuliah, error) {
	return s.matakuliahRepo.FindByID(id)
}

func (s *mataKuliahService) Update(id uint, matakuliah *model.MataKuliah) error {
	existing, err := s.matakuliahRepo.FindByID(id)
	if err != nil {
		return err
	}

	// Check if KodeMK is being changed and if new KodeMK already exists
	if existing.KodeMK != matakuliah.KodeMK {
		_, err := s.matakuliahRepo.FindByKodeMK(matakuliah.KodeMK)
		if err == nil {
			return errors.New("Kode MK already exists")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	matakuliah.ID = id
	return s.matakuliahRepo.Update(matakuliah)
}

func (s *mataKuliahService) Delete(id uint) error {
	return s.matakuliahRepo.Delete(id)
}
