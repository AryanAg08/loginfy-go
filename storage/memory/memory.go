package memory

import (
	"errors"
	"sync"
	"time"

	"github.com/AryanAg08/loginfy.go/core"
	"github.com/AryanAg08/loginfy.go/pkg/logger"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// MemoryStorage is a thread-safe in-memory implementation of the Storage interface
type MemoryStorage struct {
	users map[string]*core.User // map[userID]*User
	email map[string]string     // map[email]userID for quick email lookups
	mu    sync.RWMutex
	log   *logger.ServiceLogger
}

// New creates a new in-memory storage instance
func New() *MemoryStorage {
	return &MemoryStorage{
		users: make(map[string]*core.User),
		email: make(map[string]string),
		log:   logger.NewServiceLogger("memory-storage"),
	}
}

// CreateUser creates a new user in memory
func (m *MemoryStorage) CreateUser(user *core.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user already exists by ID
	if _, exists := m.users[user.ID]; exists {
		m.log.Warn("user creation failed: ID already exists", map[string]interface{}{
			"user_id": user.ID,
		})
		return ErrUserAlreadyExists
	}

	// Check if email already exists
	if _, exists := m.email[user.Email]; exists {
		m.log.Warn("user creation failed: email already exists", map[string]interface{}{
			"email": user.Email,
		})
		return ErrUserAlreadyExists
	}

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Store user
	m.users[user.ID] = user
	m.email[user.Email] = user.ID

	m.log.Info("user created successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return nil
}

// GetUserByEmail retrieves a user by email address
func (m *MemoryStorage) GetUserByEmail(email string) (*core.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userID, exists := m.email[email]
	if !exists {
		m.log.Debug("user not found by email", map[string]interface{}{
			"email": email,
		})
		return nil, ErrUserNotFound
	}

	user := m.users[userID]
	m.log.Debug("user retrieved by email", map[string]interface{}{
		"user_id": user.ID,
		"email":   email,
	})

	return user, nil
}

// GetUserById retrieves a user by ID
func (m *MemoryStorage) GetUserById(id string) (*core.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		m.log.Debug("user not found by ID", map[string]interface{}{
			"user_id": id,
		})
		return nil, ErrUserNotFound
	}

	m.log.Debug("user retrieved by ID", map[string]interface{}{
		"user_id": id,
	})

	return user, nil
}

// UpdateUser updates an existing user
func (m *MemoryStorage) UpdateUser(user *core.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.users[user.ID]
	if !exists {
		m.log.Warn("user update failed: user not found", map[string]interface{}{
			"user_id": user.ID,
		})
		return ErrUserNotFound
	}

	// If email changed, update the email index
	if existing.Email != user.Email {
		// Check if new email is already taken
		if _, emailExists := m.email[user.Email]; emailExists {
			m.log.Warn("user update failed: email already exists", map[string]interface{}{
				"user_id": user.ID,
				"email":   user.Email,
			})
			return ErrUserAlreadyExists
		}

		// Remove old email mapping
		delete(m.email, existing.Email)
		// Add new email mapping
		m.email[user.Email] = user.ID
	}

	// Update timestamp
	user.UpdatedAt = time.Now()

	// Update user
	m.users[user.ID] = user

	m.log.Info("user updated successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return nil
}

// DeleteUser removes a user from storage
func (m *MemoryStorage) DeleteUser(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		m.log.Warn("user deletion failed: user not found", map[string]interface{}{
			"user_id": id,
		})
		return ErrUserNotFound
	}

	// Remove email mapping
	delete(m.email, user.Email)
	// Remove user
	delete(m.users, id)

	m.log.Info("user deleted successfully", map[string]interface{}{
		"user_id": id,
	})

	return nil
}

// Count returns the total number of users in storage
func (m *MemoryStorage) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}

// Clear removes all users from storage (useful for testing)
func (m *MemoryStorage) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users = make(map[string]*core.User)
	m.email = make(map[string]string)

	m.log.Info("storage cleared", nil)
}
