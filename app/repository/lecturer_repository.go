package repository

import (
	"UASBE/app/model"
	"UASBE/database"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type LecturerRepository struct {
	db *sql.DB
}

func NewLecturerRepository() *LecturerRepository {
	return &LecturerRepository{
		db: database.GetPostgresDB(),
	}
}

func (r *LecturerRepository) Create(lecturer *model.Lecturer) error {
	lecturer.ID = uuid.New().String()
	lecturer.CreatedAt = time.Now()
	
	query := `
		INSERT INTO lecturers (id, user_id, lecturer_id, department, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	_, err := r.db.Exec(query, lecturer.ID, lecturer.UserID, lecturer.LecturerID, 
		lecturer.Department, lecturer.CreatedAt)
	
	return err
}

func (r *LecturerRepository) GetByID(id string) (*model.Lecturer, error) {
	var lecturer model.Lecturer
	
	query := `
		SELECT id, user_id, lecturer_id, department, created_at
		FROM lecturers WHERE id = $1
	`
	
	err := r.db.QueryRow(query, id).Scan(
		&lecturer.ID, &lecturer.UserID, &lecturer.LecturerID, 
		&lecturer.Department, &lecturer.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &lecturer, nil
}

func (r *LecturerRepository) GetByUserID(userID string) (*model.Lecturer, error) {
	var lecturer model.Lecturer
	
	query := `
		SELECT id, user_id, lecturer_id, department, created_at
		FROM lecturers WHERE user_id = $1
	`
	
	err := r.db.QueryRow(query, userID).Scan(
		&lecturer.ID, &lecturer.UserID, &lecturer.LecturerID, 
		&lecturer.Department, &lecturer.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &lecturer, nil
}

func (r *LecturerRepository) GetByLecturerID(lecturerID string) (*model.Lecturer, error) {
	var lecturer model.Lecturer
	
	query := `
		SELECT id, user_id, lecturer_id, department, created_at
		FROM lecturers WHERE lecturer_id = $1
	`
	
	err := r.db.QueryRow(query, lecturerID).Scan(
		&lecturer.ID, &lecturer.UserID, &lecturer.LecturerID, 
		&lecturer.Department, &lecturer.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &lecturer, nil
}

func (r *LecturerRepository) Update(lecturer *model.Lecturer) error {
	query := `
		UPDATE lecturers 
		SET user_id = $2, lecturer_id = $3, department = $4
		WHERE id = $1
	`
	
	_, err := r.db.Exec(query, lecturer.ID, lecturer.UserID, 
		lecturer.LecturerID, lecturer.Department)
	
	return err
}

func (r *LecturerRepository) Delete(id string) error {
	query := "DELETE FROM lecturers WHERE id = $1"
	_, err := r.db.Exec(query, id)
	return err
}

func (r *LecturerRepository) GetAll() ([]model.Lecturer, error) {
	var lecturers []model.Lecturer
	
	query := `
		SELECT id, user_id, lecturer_id, department, created_at
		FROM lecturers ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var lecturer model.Lecturer
		err := rows.Scan(
			&lecturer.ID, &lecturer.UserID, &lecturer.LecturerID, 
			&lecturer.Department, &lecturer.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		lecturers = append(lecturers, lecturer)
	}
	
	return lecturers, nil
}