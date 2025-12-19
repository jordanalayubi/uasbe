package repository

import (
	"UASBE/app/model"
	"UASBE/database"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type StudentRepository struct {
	db *sql.DB
}

func NewStudentRepository() *StudentRepository {
	return &StudentRepository{
		db: database.GetPostgresDB(),
	}
}

func (r *StudentRepository) Create(student *model.Student) error {
	student.ID = uuid.New().String()
	student.CreatedAt = time.Now()
	
	query := `
		INSERT INTO students (id, user_id, student_id, program_study, academic_year, advisor_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := r.db.Exec(query, student.ID, student.UserID, student.StudentID, 
		student.ProgramStudy, student.AcademicYear, student.AdvisorID, student.CreatedAt)
	
	return err
}

func (r *StudentRepository) GetByID(id string) (*model.Student, error) {
	var student model.Student
	
	query := `
		SELECT id, user_id, student_id, program_study, academic_year, 
		       advisor_id, created_at
		FROM students WHERE id = $1
	`
	
	var advisorID sql.NullString
	err := r.db.QueryRow(query, id).Scan(
		&student.ID, &student.UserID, &student.StudentID, &student.ProgramStudy,
		&student.AcademicYear, &advisorID, &student.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Handle NULL advisor_id properly
	if advisorID.Valid {
		student.AdvisorID = advisorID.String
	} else {
		student.AdvisorID = ""
	}
	
	if err != nil {
		return nil, err
	}
	
	return &student, nil
}

func (r *StudentRepository) GetByUserID(userID string) (*model.Student, error) {
	var student model.Student
	
	query := `
		SELECT id, user_id, student_id, program_study, academic_year, 
		       advisor_id, created_at
		FROM students WHERE user_id = $1
	`
	
	var advisorID sql.NullString
	err := r.db.QueryRow(query, userID).Scan(
		&student.ID, &student.UserID, &student.StudentID, &student.ProgramStudy,
		&student.AcademicYear, &advisorID, &student.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Handle NULL advisor_id properly
	if advisorID.Valid {
		student.AdvisorID = advisorID.String
	} else {
		student.AdvisorID = ""
	}
	
	if err != nil {
		return nil, err
	}
	
	return &student, nil
}

func (r *StudentRepository) GetByStudentID(studentID string) (*model.Student, error) {
	var student model.Student
	
	query := `
		SELECT id, user_id, student_id, program_study, academic_year, advisor_id, created_at
		FROM students WHERE student_id = $1
	`
	
	var advisorID sql.NullString
	err := r.db.QueryRow(query, studentID).Scan(
		&student.ID, &student.UserID, &student.StudentID, &student.ProgramStudy,
		&student.AcademicYear, &advisorID, &student.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Handle NULL advisor_id properly
	if advisorID.Valid {
		student.AdvisorID = advisorID.String
	} else {
		student.AdvisorID = ""
	}
	
	return &student, nil
}

func (r *StudentRepository) GetByAdvisorID(advisorID string) ([]model.Student, error) {
	var students []model.Student
	
	query := `
		SELECT id, user_id, student_id, program_study, academic_year, advisor_id, created_at
		FROM students WHERE advisor_id = $1 ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, advisorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var student model.Student
		err := rows.Scan(
			&student.ID, &student.UserID, &student.StudentID, &student.ProgramStudy,
			&student.AcademicYear, &student.AdvisorID, &student.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		students = append(students, student)
	}
	
	return students, nil
}

func (r *StudentRepository) Update(student *model.Student) error {
	query := `
		UPDATE students 
		SET user_id = $2, student_id = $3, program_study = $4, 
		    academic_year = $5, advisor_id = $6
		WHERE id = $1
	`
	
	_, err := r.db.Exec(query, student.ID, student.UserID, student.StudentID,
		student.ProgramStudy, student.AcademicYear, student.AdvisorID)
	
	return err
}

func (r *StudentRepository) Delete(id string) error {
	query := "DELETE FROM students WHERE id = $1"
	_, err := r.db.Exec(query, id)
	return err
}

func (r *StudentRepository) GetAll() ([]model.Student, error) {
	var students []model.Student
	
	query := `
		SELECT id, user_id, student_id, program_study, academic_year, advisor_id, created_at
		FROM students ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var student model.Student
		var advisorID sql.NullString
		
		err := rows.Scan(
			&student.ID, &student.UserID, &student.StudentID, &student.ProgramStudy,
			&student.AcademicYear, &advisorID, &student.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		// Handle NULL advisor_id properly
		if advisorID.Valid {
			student.AdvisorID = advisorID.String
		} else {
			student.AdvisorID = ""
		}
		
		students = append(students, student)
	}
	
	return students, nil
}
// FR-010: Get students by multiple user IDs for batch lookup
func (r *StudentRepository) GetStudentsByUserIDs(userIDs []string) ([]model.Student, error) {
	var students []model.Student
	
	if len(userIDs) == 0 {
		return students, nil
	}
	
	// Filter out empty strings to avoid UUID parsing errors
	var validUserIDs []string
	for _, id := range userIDs {
		if id != "" && len(id) > 0 {
			validUserIDs = append(validUserIDs, id)
		}
	}
	
	if len(validUserIDs) == 0 {
		return students, nil
	}
	
	// Create placeholders for IN clause
	placeholders := make([]string, len(validUserIDs))
	args := make([]interface{}, len(validUserIDs))
	for i, id := range validUserIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	
	query := fmt.Sprintf(`
		SELECT id, user_id, student_id, program_study, academic_year, 
		       advisor_id, created_at
		FROM students WHERE user_id IN (%s)
		ORDER BY created_at DESC
	`, strings.Join(placeholders, ","))
	
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var student model.Student
		var advisorID sql.NullString
		
		err := rows.Scan(
			&student.ID, &student.UserID, &student.StudentID, &student.ProgramStudy,
			&student.AcademicYear, &advisorID, &student.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		// Handle NULL advisor_id properly
		if advisorID.Valid {
			student.AdvisorID = advisorID.String
		} else {
			student.AdvisorID = ""
		}
		
		students = append(students, student)
	}
	
	return students, nil
}