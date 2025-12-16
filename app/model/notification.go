package model

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id" json:"user_id"` // PostgreSQL UUID as string
	Type        string             `bson:"type" json:"type"`       // achievement_submitted, achievement_verified, etc.
	Title       string             `bson:"title" json:"title"`
	Message     string             `bson:"message" json:"message"`
	Data        map[string]interface{} `bson:"data,omitempty" json:"data,omitempty"`
	IsRead      bool               `bson:"is_read" json:"is_read"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	ReadAt      *time.Time         `bson:"read_at,omitempty" json:"read_at,omitempty"`
}