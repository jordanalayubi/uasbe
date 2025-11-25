package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"
	"UASBE/app/utils"
	"errors"

	"gorm.io/gorm"
)

type MahasiswaService interface {
	Create(mahasiswa *model.Mahasiswa, username, password string) error
	GetAll() ([]model.Mahasiswa, error)
	GetByID(id uint) (*model.Mahasiswa, error)
	Update(id uint, mahasiswa *model.Mahasiswa) error
	Delete(id uint) error
}

type mahasiswaService struct {
	mahasiswaRepo repository.MahasiswaRepository
	userRepo      repository.UserRepository
}

func NewMahasiswaService(mahasiswaRepo repository.MahasiswaRepository, userRepo repository.UserRepository) MahasiswaService {
	return &mahasiswaService{
		mahasiswaRepo: mahasiswaRepo,
		userRepo:      userRepo,
	}
}

func (s *mahasiswaService) Create(mahasiswa *model.Mahasiswa, username, password string) error {
	// Check if NIM already exists
	_, err := s.mahasiswaRepo.FindByNIM(mahasiswa.NIM)
	if err == nil {
		return errors.New("NIM already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Create user account
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	user := &model.User{
		Username: username,
		Password: hashedPassword,
		Role:     "mahasiswa",
	}

	if err := s.userRepo.Create(user); err != nil {
		return err
	}

	mahasiswa.UserID = user.ID
	return s.mahasiswaRepo.Create(mahasiswa)
}

func (s *mahasiswaService) GetAll() ([]model.Mahasiswa, error) {
	return s.mahasiswaRepo.FindAll()
}

func (s *mahasiswaService) GetByID(id uint) (*model.Mahasiswa, error) {
	return s.mahasiswaRepo.FindByID(id)
}

func (s *mahasiswaService) Update(id uint, mahasiswa *model.Mahasiswa) error {
	existing, err := s.mahasiswaRepo.FindByID(id)
	if err != nil {
		return err
	}

	// Check if NIM is being changed and if new NIM already exists
	if existing.NIM != mahasiswa.NIM {
		_, err := s.mahasiswaRepo.FindByNIM(mahasiswa.NIM)
		if err == nil {
			return errors.New("NIM already exists")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	mahasiswa.ID = id
	mahasiswa.UserID = existing.UserID
	return s.mahasiswaRepo.Update(mahasiswa)
}

func (s *mahasiswaService) Delete(id uint) error {
	return s.mahasiswaRepo.Delete(id)
}
