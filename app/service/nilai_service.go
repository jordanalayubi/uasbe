package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"
)

type NilaiService interface {
	Create(nilai *model.Nilai) error
	GetAll() ([]model.Nilai, error)
	GetByID(id uint) (*model.Nilai, error)
	GetByMahasiswaID(mahasiswaID uint) ([]model.Nilai, error)
	Update(id uint, nilai *model.Nilai) error
	Delete(id uint) error
}

type nilaiService struct {
	nilaiRepo      repository.NilaiRepository
	mahasiswaRepo  repository.MahasiswaRepository
	matakuliahRepo repository.MataKuliahRepository
}

func NewNilaiService(
	nilaiRepo repository.NilaiRepository,
	mahasiswaRepo repository.MahasiswaRepository,
	matakuliahRepo repository.MataKuliahRepository,
) NilaiService {
	return &nilaiService{
		nilaiRepo:      nilaiRepo,
		mahasiswaRepo:  mahasiswaRepo,
		matakuliahRepo: matakuliahRepo,
	}
}

func (s *nilaiService) Create(nilai *model.Nilai) error {
	// Validate mahasiswa exists
	_, err := s.mahasiswaRepo.FindByID(nilai.MahasiswaID)
	if err != nil {
		return err
	}

	// Validate matakuliah exists
	_, err = s.matakuliahRepo.FindByID(nilai.MataKuliahID)
	if err != nil {
		return err
	}

	return s.nilaiRepo.Create(nilai)
}

func (s *nilaiService) GetAll() ([]model.Nilai, error) {
	return s.nilaiRepo.FindAll()
}

func (s *nilaiService) GetByID(id uint) (*model.Nilai, error) {
	return s.nilaiRepo.FindByID(id)
}

func (s *nilaiService) GetByMahasiswaID(mahasiswaID uint) ([]model.Nilai, error) {
	return s.nilaiRepo.FindByMahasiswaID(mahasiswaID)
}

func (s *nilaiService) Update(id uint, nilai *model.Nilai) error {
	// Validate mahasiswa exists
	_, err := s.mahasiswaRepo.FindByID(nilai.MahasiswaID)
	if err != nil {
		return err
	}

	// Validate matakuliah exists
	_, err = s.matakuliahRepo.FindByID(nilai.MataKuliahID)
	if err != nil {
		return err
	}

	nilai.ID = id
	return s.nilaiRepo.Update(nilai)
}

func (s *nilaiService) Delete(id uint) error {
	return s.nilaiRepo.Delete(id)
}
