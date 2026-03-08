package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// MongoDBStorage is a MongoDB implementation of the Storage interface.
type MongoDBStorage struct {
	client     *mongo.Client
	collection *mongo.Collection
	log        *logger.ServiceLogger
}

// New creates a new MongoDB storage instance by connecting to the given connection string.
func New(connString string, database string) (*MongoDBStorage, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(connString))
	if err != nil {
		return nil, err
	}

	s := &MongoDBStorage{
		client:     client,
		collection: client.Database(database).Collection("users"),
		log:        logger.NewServiceLogger("mongodb-storage"),
	}

	if err := s.ensureIndexes(); err != nil {
		client.Disconnect(context.Background())
		return nil, err
	}

	s.log.Info("mongodb storage initialized", map[string]interface{}{
		"database": database,
	})
	return s, nil
}

// NewFromClient creates a MongoDBStorage from an existing mongo client.
func NewFromClient(client *mongo.Client, database string) *MongoDBStorage {
	return &MongoDBStorage{
		client:     client,
		collection: client.Database(database).Collection("users"),
		log:        logger.NewServiceLogger("mongodb-storage"),
	}
}

func (s *MongoDBStorage) ensureIndexes() error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := s.collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		s.log.Error("failed to create email index", map[string]interface{}{
			"error": err.Error(),
		})
	}
	return err
}

// Connect verifies the MongoDB connection is alive.
func (s *MongoDBStorage) Connect() error {
	return s.client.Ping(context.Background(), nil)
}

// Close disconnects from MongoDB.
func (s *MongoDBStorage) Close() error {
	return s.client.Disconnect(context.Background())
}

// Ping checks if MongoDB is reachable.
func (s *MongoDBStorage) Ping() error {
	return s.client.Ping(context.Background(), nil)
}

// CreateUser creates a new user in MongoDB.
func (s *MongoDBStorage) CreateUser(user *core.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := s.collection.InsertOne(context.Background(), user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			s.log.Warn("user creation failed: duplicate", map[string]interface{}{
				"user_id": user.ID,
				"email":   user.Email,
			})
			return ErrUserAlreadyExists
		}
		s.log.Warn("user creation failed", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return err
	}

	s.log.Info("user created successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})
	return nil
}

// GetUserByEmail retrieves a user by email address.
func (s *MongoDBStorage) GetUserByEmail(email string) (*core.User, error) {
	var user core.User
	err := s.collection.FindOne(context.Background(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserById retrieves a user by ID.
func (s *MongoDBStorage) GetUserById(id string) (*core.User, error) {
	var user core.User
	err := s.collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates an existing user in MongoDB.
func (s *MongoDBStorage) UpdateUser(user *core.User) error {
	user.UpdatedAt = time.Now()

	result, err := s.collection.ReplaceOne(
		context.Background(),
		bson.M{"_id": user.ID},
		user,
	)
	if err != nil {
		s.log.Warn("user update failed", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return err
	}
	if result.MatchedCount == 0 {
		return ErrUserNotFound
	}

	s.log.Info("user updated successfully", map[string]interface{}{
		"user_id": user.ID,
	})
	return nil
}

// DeleteUser removes a user from MongoDB.
func (s *MongoDBStorage) DeleteUser(id string) error {
	result, err := s.collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		s.log.Warn("user deletion failed", map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		})
		return err
	}
	if result.DeletedCount == 0 {
		return ErrUserNotFound
	}

	s.log.Info("user deleted successfully", map[string]interface{}{
		"user_id": id,
	})
	return nil
}
