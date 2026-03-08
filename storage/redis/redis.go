package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

const (
	userKeyPrefix = "user:"
	emailIndexKey = "user:email_index"
)

// RedisStorage is a Redis implementation of the Storage interface.
type RedisStorage struct {
	client *redis.Client
	log    *logger.ServiceLogger
}

// New creates a new Redis storage instance with the given options.
func New(opts *redis.Options) *RedisStorage {
	return &RedisStorage{
		client: redis.NewClient(opts),
		log:    logger.NewServiceLogger("redis-storage"),
	}
}

// NewFromClient creates a RedisStorage from an existing redis client.
func NewFromClient(client *redis.Client) *RedisStorage {
	return &RedisStorage{
		client: client,
		log:    logger.NewServiceLogger("redis-storage"),
	}
}

// Connect verifies the Redis connection is alive.
func (s *RedisStorage) Connect() error {
	return s.client.Ping(context.Background()).Err()
}

// Close closes the Redis connection.
func (s *RedisStorage) Close() error {
	return s.client.Close()
}

// Ping checks if Redis is reachable.
func (s *RedisStorage) Ping() error {
	return s.client.Ping(context.Background()).Err()
}

// CreateUser creates a new user in Redis.
func (s *RedisStorage) CreateUser(user *core.User) error {
	ctx := context.Background()

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Check if email already exists
	exists, err := s.client.HExists(ctx, emailIndexKey, user.Email).Result()
	if err != nil {
		return err
	}
	if exists {
		s.log.Warn("user creation failed: email already exists", map[string]interface{}{
			"email": user.Email,
		})
		return ErrUserAlreadyExists
	}

	// Check if user ID already exists
	userKey := userKeyPrefix + user.ID
	idExists, err := s.client.Exists(ctx, userKey).Result()
	if err != nil {
		return err
	}
	if idExists > 0 {
		s.log.Warn("user creation failed: ID already exists", map[string]interface{}{
			"user_id": user.ID,
		})
		return ErrUserAlreadyExists
	}

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	// Use a transaction to atomically set both the user data and email index
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, userKey, data, 0)
	pipe.HSet(ctx, emailIndexKey, user.Email, user.ID)
	if _, err := pipe.Exec(ctx); err != nil {
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
func (s *RedisStorage) GetUserByEmail(email string) (*core.User, error) {
	ctx := context.Background()

	userID, err := s.client.HGet(ctx, emailIndexKey, email).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return s.GetUserById(userID)
}

// GetUserById retrieves a user by ID.
func (s *RedisStorage) GetUserById(id string) (*core.User, error) {
	ctx := context.Background()
	userKey := userKeyPrefix + id

	data, err := s.client.Get(ctx, userKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var user core.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}
	// Password is excluded from JSON via json:"-", so we store it separately
	// We include password in the stored JSON by using a wrapper
	return s.getUserFull(id)
}

type userWithPassword struct {
	core.User
	Password string `json:"password"`
}

func (s *RedisStorage) getUserFull(id string) (*core.User, error) {
	ctx := context.Background()
	userKey := userKeyPrefix + id

	data, err := s.client.Get(ctx, userKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var uwp userWithPassword
	if err := json.Unmarshal([]byte(data), &uwp); err != nil {
		return nil, err
	}
	uwp.User.Password = uwp.Password
	return &uwp.User, nil
}

func (s *RedisStorage) marshalUser(user *core.User) ([]byte, error) {
	uwp := userWithPassword{
		User:     *user,
		Password: user.Password,
	}
	return json.Marshal(uwp)
}

// UpdateUser updates an existing user in Redis.
func (s *RedisStorage) UpdateUser(user *core.User) error {
	ctx := context.Background()
	userKey := userKeyPrefix + user.ID

	// Check if user exists
	existing, err := s.getUserFull(user.ID)
	if err != nil {
		return err
	}

	user.UpdatedAt = time.Now()

	data, err := s.marshalUser(user)
	if err != nil {
		return err
	}

	pipe := s.client.TxPipeline()

	// If email changed, update the email index
	if existing.Email != user.Email {
		// Check if new email is already taken
		emailExists, err := s.client.HExists(ctx, emailIndexKey, user.Email).Result()
		if err != nil {
			return err
		}
		if emailExists {
			return ErrUserAlreadyExists
		}
		pipe.HDel(ctx, emailIndexKey, existing.Email)
		pipe.HSet(ctx, emailIndexKey, user.Email, user.ID)
	}

	pipe.Set(ctx, userKey, data, 0)
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Warn("user update failed", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return err
	}

	s.log.Info("user updated successfully", map[string]interface{}{
		"user_id": user.ID,
	})
	return nil
}

// DeleteUser removes a user from Redis.
func (s *RedisStorage) DeleteUser(id string) error {
	ctx := context.Background()

	// Get user to find email for index cleanup
	existing, err := s.getUserFull(id)
	if err != nil {
		return err
	}

	userKey := userKeyPrefix + id
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, userKey)
	pipe.HDel(ctx, emailIndexKey, existing.Email)
	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Warn("user deletion failed", map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		})
		return err
	}

	s.log.Info("user deleted successfully", map[string]interface{}{
		"user_id": id,
	})
	return nil
}
