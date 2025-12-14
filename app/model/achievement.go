package model

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Achievement struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID   string             `bson:"student_id" json:"student_id"` // PostgreSQL UUID as string
	ObjectID    string             `bson:"object_id" json:"object_id"`
	StudentInfo string             `bson:"student_info" json:"student_info"`
	Category    string             `bson:"category" json:"category"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	
	// Field dinamis berdasarkan tipe prestasi
	Details AchievementDetails `bson:"details" json:"details"`
	
	CustomFields []CustomField `bson:"custom_fields,omitempty" json:"custom_fields,omitempty"`
	
	Attachments []Attachment `bson:"attachments" json:"attachments"`
	
	Tags []string `bson:"tags" json:"tags"`
	
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type AchievementDetails struct {
	// Untuk competition
	CompetitionName  string `bson:"competition_name,omitempty" json:"competition_name,omitempty"`
	CompetitionLevel string `bson:"competition_level,omitempty" json:"competition_level,omitempty"`
	Rank             int    `bson:"rank,omitempty" json:"rank,omitempty"`
	Medal            string `bson:"medal,omitempty" json:"medal,omitempty"`
	
	// Untuk publication
	PublicationType   string `bson:"publication_type,omitempty" json:"publication_type,omitempty"`
	PublicationTitle  string `bson:"publication_title,omitempty" json:"publication_title,omitempty"`
	PublicationJournal string `bson:"publication_journal,omitempty" json:"publication_journal,omitempty"`
	Publisher         string `bson:"publisher,omitempty" json:"publisher,omitempty"`
	ISSN              string `bson:"issn,omitempty" json:"issn,omitempty"`
	
	// Untuk organization
	OrganizationName string    `bson:"organization_name,omitempty" json:"organization_name,omitempty"`
	Position         string    `bson:"position,omitempty" json:"position,omitempty"`
	Period           struct {
		Start time.Time `bson:"start,omitempty" json:"start,omitempty"`
		End   time.Time `bson:"end,omitempty" json:"end,omitempty"`
	} `bson:"period,omitempty" json:"period,omitempty"`
	
	// Untuk certification
	CertificationName   string    `bson:"certification_name,omitempty" json:"certification_name,omitempty"`
	IssuedBy           string    `bson:"issued_by,omitempty" json:"issued_by,omitempty"`
	CertificationNumber string    `bson:"certification_number,omitempty" json:"certification_number,omitempty"`
	ValidUntil         time.Time `bson:"valid_until,omitempty" json:"valid_until,omitempty"`
	
	// Field umum yang bisa ada
	EventDate  time.Time `bson:"event_date,omitempty" json:"event_date,omitempty"`
	Location   string    `bson:"location,omitempty" json:"location,omitempty"`
	Organizer  string    `bson:"organizer,omitempty" json:"organizer,omitempty"`
	Score      int       `bson:"score,omitempty" json:"score,omitempty"`
}

type CustomField struct {
	Name  string      `bson:"name" json:"name"`
	Value interface{} `bson:"value" json:"value"`
}