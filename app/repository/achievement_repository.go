package repository

import (
	"UASBE/app/model"
	"UASBE/database"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AchievementRepository struct {
	collection          *mongo.Collection
	referenceCollection *mongo.Collection
}

func NewAchievementRepository() *AchievementRepository {
	return &AchievementRepository{
		collection:          database.GetMongoCollection("achievements"),
		referenceCollection: database.GetMongoCollection("achievement_references"),
	}
}

func (r *AchievementRepository) Create(achievement *model.Achievement) error {
	achievement.CreatedAt = time.Now()
	achievement.UpdatedAt = time.Now()
	
	result, err := r.collection.InsertOne(context.Background(), achievement)
	if err != nil {
		return err
	}
	
	achievement.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *AchievementRepository) CreateReference(ref *model.AchievementReference) error {
	ref.CreatedAt = time.Now()
	ref.UpdatedAt = time.Now()
	
	result, err := r.referenceCollection.InsertOne(context.Background(), ref)
	if err != nil {
		return err
	}
	
	ref.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *AchievementRepository) GetByID(id primitive.ObjectID) (*model.Achievement, error) {
	var achievement model.Achievement
	err := r.collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&achievement)
	if err != nil {
		return nil, err
	}
	return &achievement, nil
}

func (r *AchievementRepository) GetReferenceByID(id primitive.ObjectID) (*model.AchievementReference, error) {
	var ref model.AchievementReference
	err := r.referenceCollection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&ref)
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (r *AchievementRepository) GetByStudentID(studentID string) ([]model.Achievement, error) {
	var achievements []model.Achievement
	
	cursor, err := r.collection.Find(context.Background(), bson.M{"student_id": studentID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}

func (r *AchievementRepository) GetReferencesByStudentID(studentID string) ([]model.AchievementReference, error) {
	var references []model.AchievementReference
	
	cursor, err := r.referenceCollection.Find(context.Background(), bson.M{"student_id": studentID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &references)
	return references, err
}

func (r *AchievementRepository) GetReferencesByStatus(status string) ([]model.AchievementReference, error) {
	var references []model.AchievementReference
	
	cursor, err := r.referenceCollection.Find(context.Background(), bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &references)
	return references, err
}

func (r *AchievementRepository) Update(achievement *model.Achievement) error {
	achievement.UpdatedAt = time.Now()
	
	filter := bson.M{"_id": achievement.ID}
	update := bson.M{"$set": achievement}
	
	_, err := r.collection.UpdateOne(context.Background(), filter, update)
	return err
}

func (r *AchievementRepository) UpdateReference(ref *model.AchievementReference) error {
	ref.UpdatedAt = time.Now()
	
	filter := bson.M{"_id": ref.ID}
	update := bson.M{"$set": ref}
	
	_, err := r.referenceCollection.UpdateOne(context.Background(), filter, update)
	return err
}

func (r *AchievementRepository) Delete(id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(context.Background(), bson.M{"_id": id})
	return err
}

func (r *AchievementRepository) DeleteReference(id primitive.ObjectID) error {
	_, err := r.referenceCollection.DeleteOne(context.Background(), bson.M{"_id": id})
	return err
}

func (r *AchievementRepository) GetAll(limit, offset int) ([]model.Achievement, error) {
	var achievements []model.Achievement
	
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	
	cursor, err := r.collection.Find(context.Background(), bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}

func (r *AchievementRepository) SearchByCategory(category string) ([]model.Achievement, error) {
	var achievements []model.Achievement
	
	cursor, err := r.collection.Find(context.Background(), bson.M{"category": category})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}

func (r *AchievementRepository) SearchByTags(tags []string) ([]model.Achievement, error) {
	var achievements []model.Achievement
	
	filter := bson.M{"tags": bson.M{"$in": tags}}
	cursor, err := r.collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}