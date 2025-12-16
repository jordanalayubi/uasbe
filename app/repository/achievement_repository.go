package repository

import (
	"UASBE/app/model"
	"UASBE/database"
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
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
	
	// Include soft deleted achievements for internal operations
	err := r.collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&achievement)
	if err != nil {
		return nil, err
	}
	return &achievement, nil
}

func (r *AchievementRepository) GetByIDActive(id primitive.ObjectID) (*model.Achievement, error) {
	var achievement model.Achievement
	
	// Filter out soft deleted achievements for public access
	filter := bson.M{
		"_id": id,
		"deleted_at": bson.M{"$exists": false},
	}
	
	err := r.collection.FindOne(context.Background(), filter).Decode(&achievement)
	if err != nil {
		return nil, err
	}
	return &achievement, nil
}

func (r *AchievementRepository) GetByObjectID(objectID string) (*model.Achievement, error) {
	var achievement model.Achievement
	
	// Include soft deleted achievements for internal operations
	err := r.collection.FindOne(context.Background(), bson.M{"object_id": objectID}).Decode(&achievement)
	if err != nil {
		return nil, err
	}
	return &achievement, nil
}

func (r *AchievementRepository) GetByObjectIDActive(objectID string) (*model.Achievement, error) {
	var achievement model.Achievement
	
	// Filter out soft deleted achievements for public access
	filter := bson.M{
		"object_id": objectID,
		"deleted_at": bson.M{"$exists": false},
	}
	
	err := r.collection.FindOne(context.Background(), filter).Decode(&achievement)
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
	
	// Filter out soft deleted achievements
	filter := bson.M{
		"student_id": studentID,
		"deleted_at": bson.M{"$exists": false},
	}
	
	cursor, err := r.collection.Find(context.Background(), filter)
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
	
	// Filter out soft deleted achievements
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	
	cursor, err := r.collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}

func (r *AchievementRepository) SearchByCategory(category string) ([]model.Achievement, error) {
	var achievements []model.Achievement
	
	// Filter out soft deleted achievements
	filter := bson.M{
		"category": category,
		"deleted_at": bson.M{"$exists": false},
	}
	
	cursor, err := r.collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}

func (r *AchievementRepository) SearchByTags(tags []string) ([]model.Achievement, error) {
	var achievements []model.Achievement
	
	// Filter out soft deleted achievements
	filter := bson.M{
		"tags": bson.M{"$in": tags},
		"deleted_at": bson.M{"$exists": false},
	}
	
	cursor, err := r.collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}

// FR-006: Get achievements by multiple student IDs with pagination
func (r *AchievementRepository) GetByStudentIDs(studentIDs []string, limit, offset int) ([]model.Achievement, error) {
	var achievements []model.Achievement
	
	// Filter out soft deleted achievements and filter by student IDs
	filter := bson.M{
		"student_id": bson.M{"$in": studentIDs},
		"deleted_at": bson.M{"$exists": false},
	}
	
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	// Sort by created_at descending (newest first)
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	
	cursor, err := r.collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}

// FR-006: Get achievement references by multiple student IDs with pagination
func (r *AchievementRepository) GetReferencesByStudentIDs(studentIDs []string, limit, offset int) ([]model.AchievementReference, error) {
	var references []model.AchievementReference
	
	filter := bson.M{"student_id": bson.M{"$in": studentIDs}}
	
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	// Sort by created_at descending (newest first)
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	
	cursor, err := r.referenceCollection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &references)
	return references, err
}

// FR-006: Count achievements by student IDs for pagination
func (r *AchievementRepository) CountByStudentIDs(studentIDs []string) (int64, error) {
	filter := bson.M{
		"student_id": bson.M{"$in": studentIDs},
		"deleted_at": bson.M{"$exists": false},
	}
	
	count, err := r.collection.CountDocuments(context.Background(), filter)
	return count, err
}

// FR-010: Get all achievement references with filters and pagination
func (r *AchievementRepository) GetAllReferencesWithFilters(page, limit int, status, studentID string) ([]model.AchievementReference, int, error) {
	var references []model.AchievementReference
	
	// Build filter
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	if studentID != "" {
		filter["student_id"] = studentID
	}
	
	// Calculate offset
	offset := (page - 1) * limit
	
	// Get total count
	total, err := r.referenceCollection.CountDocuments(context.Background(), filter)
	if err != nil {
		return nil, 0, err
	}
	
	// Get references with pagination
	opts := options.Find()
	opts.SetLimit(int64(limit))
	opts.SetSkip(int64(offset))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}}) // Sort by newest first
	
	cursor, err := r.referenceCollection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &references)
	return references, int(total), err
}

// FR-010: Get achievements by IDs with filters and sorting
func (r *AchievementRepository) GetAchievementsByIDsWithFilters(achievementIDs []string, category, sortBy, sortOrder string) ([]model.Achievement, error) {
	var achievements []model.Achievement
	
	// Convert string IDs to ObjectIDs
	var objectIDs []primitive.ObjectID
	for _, id := range achievementIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue // Skip invalid IDs
		}
		objectIDs = append(objectIDs, objectID)
	}
	
	if len(objectIDs) == 0 {
		return achievements, nil
	}
	
	// Build filter
	filter := bson.M{
		"_id": bson.M{"$in": objectIDs},
		"deleted_at": bson.M{"$exists": false}, // Exclude soft deleted
	}
	
	if category != "" {
		filter["category"] = category
	}
	
	// Build sort options
	opts := options.Find()
	sortValue := -1 // Default descending
	if sortOrder == "asc" {
		sortValue = 1
	}
	
	switch sortBy {
	case "title":
		opts.SetSort(bson.D{{Key: "title", Value: sortValue}})
	case "category":
		opts.SetSort(bson.D{{Key: "category", Value: sortValue}})
	case "updated_at":
		opts.SetSort(bson.D{{Key: "updated_at", Value: sortValue}})
	default: // created_at
		opts.SetSort(bson.D{{Key: "created_at", Value: sortValue}})
	}
	
	cursor, err := r.collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())
	
	err = cursor.All(context.Background(), &achievements)
	return achievements, err
}
// FR-010: Get achievement statistics for admin dashboard
func (r *AchievementRepository) GetAchievementStatistics() (fiber.Map, error) {
	// Get status statistics from references
	statusPipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$status",
				"count": bson.M{"$sum": 1},
			},
		},
	}
	
	statusCursor, err := r.referenceCollection.Aggregate(context.Background(), statusPipeline)
	if err != nil {
		return nil, err
	}
	defer statusCursor.Close(context.Background())
	
	statusStats := make(map[string]int)
	for statusCursor.Next(context.Background()) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := statusCursor.Decode(&result); err != nil {
			continue
		}
		statusStats[result.ID] = result.Count
	}
	
	// Get category statistics from achievements
	categoryPipeline := []bson.M{
		{
			"$match": bson.M{
				"deleted_at": bson.M{"$exists": false},
			},
		},
		{
			"$group": bson.M{
				"_id":   "$category",
				"count": bson.M{"$sum": 1},
			},
		},
	}
	
	categoryCursor, err := r.collection.Aggregate(context.Background(), categoryPipeline)
	if err != nil {
		return nil, err
	}
	defer categoryCursor.Close(context.Background())
	
	categoryStats := make(map[string]int)
	for categoryCursor.Next(context.Background()) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := categoryCursor.Decode(&result); err != nil {
			continue
		}
		categoryStats[result.ID] = result.Count
	}
	
	return fiber.Map{
		"by_status":   statusStats,
		"by_category": categoryStats,
	}, nil
}
// GetReferenceByAchievementID - Get reference by achievement ID
func (r *AchievementRepository) GetReferenceByAchievementID(achievementID string) (*model.AchievementReference, error) {
	var reference model.AchievementReference
	
	filter := bson.M{"achievement_id": achievementID}
	
	err := r.referenceCollection.FindOne(context.Background(), filter).Decode(&reference)
	if err != nil {
		return nil, err
	}
	
	return &reference, nil
}
