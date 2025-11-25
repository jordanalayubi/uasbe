package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"
	"UASBE/app/utils"
	"errors"

	"gorm.io/gorm"
)

type AuthService interface {
	Login(username, password string) (*model.LoginResponse, error)
}

type authService struct {
	userRepo repository.UserRepository
}

func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo: userRepo}
}

func (s *authService) Login(username, password string) (*model.LoginResponse, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}

	if err := utils.CheckPassword(user.Password, password); err != nil {
		return nil, errors.New("invalid username or password")
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		Token:    token,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}
