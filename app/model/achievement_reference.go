package model

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AchievementReference struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID    string             `bson:"student_id" json:"student_id"` // PostgreSQL UUID as string
	AchievementID string            `bson:"achievement_id" json:"achievement_id"`
	Status       string             `bson:"status" json:"status"` // draft, submitted, verified, rejected
	SubmittedAt  time.Time          `bson:"submitted_at,omitempty" json:"submitted_at,omitempty"`
	VerifiedAt   time.Time          `bson:"verified_at,omitempty" json:"verified_at,omitempty"`
	VerifiedBy   string             `bson:"verified_by,omitempty" json:"verified_by,omitempty"` // PostgreSQL UUID as string
	RejectionNote string            `bson:"rejection_note,omitempty" json:"rejection_note,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}