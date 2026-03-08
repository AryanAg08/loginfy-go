package cassandra

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gocql/gocql"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// CassandraStorage is a Cassandra implementation of the Storage interface.
type CassandraStorage struct {
	session  *gocql.Session
	keyspace string
	log      *logger.ServiceLogger
}

// New creates a new Cassandra storage instance by connecting to the given hosts and keyspace.
func New(hosts []string, keyspace string) (*CassandraStorage, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = "system"
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	s := &CassandraStorage{
		session:  session,
		keyspace: keyspace,
		log:      logger.NewServiceLogger("cassandra-storage"),
	}

	if err := s.ensureKeyspace(); err != nil {
		session.Close()
		return nil, err
	}

	// Reconnect with the target keyspace
	session.Close()
	cluster.Keyspace = keyspace
	session, err = cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	s.session = session

	if err := s.autoMigrate(); err != nil {
		session.Close()
		return nil, err
	}

	s.log.Info("cassandra storage initialized", map[string]interface{}{
		"keyspace": keyspace,
	})
	return s, nil
}

// NewFromSession creates a CassandraStorage from an existing gocql session.
func NewFromSession(session *gocql.Session) *CassandraStorage {
	return &CassandraStorage{
		session: session,
		log:     logger.NewServiceLogger("cassandra-storage"),
	}
}

func (s *CassandraStorage) ensureKeyspace() error {
	query := `CREATE KEYSPACE IF NOT EXISTS ` + s.keyspace + ` WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}`
	return s.session.Query(query).Exec()
}

func (s *CassandraStorage) autoMigrate() error {
	query := `CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT,
		password TEXT,
		roles TEXT,
		metadata TEXT,
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	)`
	if err := s.session.Query(query).Exec(); err != nil {
		s.log.Error("failed to create users table", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	// Create a secondary index on email for lookups
	indexQuery := `CREATE INDEX IF NOT EXISTS users_email_idx ON users (email)`
	if err := s.session.Query(indexQuery).Exec(); err != nil {
		s.log.Error("failed to create email index", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	return nil
}

// Connect is a no-op since the session is already established during New().
func (s *CassandraStorage) Connect() error {
	if s.session.Closed() {
		return errors.New("cassandra session is closed")
	}
	return nil
}

// Close closes the Cassandra session.
func (s *CassandraStorage) Close() {
	s.session.Close()
	s.log.Info("cassandra connection closed", nil)
}

// CreateUser creates a new user in Cassandra.
func (s *CassandraStorage) CreateUser(user *core.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Check if email already exists
	var existingID string
	if err := s.session.Query(`SELECT id FROM users WHERE email = ? LIMIT 1`, user.Email).Scan(&existingID); err == nil {
		s.log.Warn("user creation failed: email already exists", map[string]interface{}{
			"email": user.Email,
		})
		return ErrUserAlreadyExists
	}

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(user.Metadata)
	if err != nil {
		return err
	}

	// Use IF NOT EXISTS for atomic insert
	applied, err := s.session.Query(
		`INSERT INTO users (id, email, password, roles, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?) IF NOT EXISTS`,
		user.ID, user.Email, user.Password, string(rolesJSON), string(metadataJSON), user.CreatedAt, user.UpdatedAt,
	).ScanCAS()
	if err != nil {
		s.log.Warn("user creation failed", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return err
	}
	if !applied {
		return ErrUserAlreadyExists
	}

	s.log.Info("user created successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})
	return nil
}

// GetUserByEmail retrieves a user by email address.
func (s *CassandraStorage) GetUserByEmail(email string) (*core.User, error) {
	var user core.User
	var rolesJSON, metadataJSON string

	err := s.session.Query(
		`SELECT id, email, password, roles, metadata, created_at, updated_at FROM users WHERE email = ? LIMIT 1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Password, &rolesJSON, &metadataJSON, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err := s.unmarshalFields(&user, rolesJSON, metadataJSON); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserById retrieves a user by ID.
func (s *CassandraStorage) GetUserById(id string) (*core.User, error) {
	var user core.User
	var rolesJSON, metadataJSON string

	err := s.session.Query(
		`SELECT id, email, password, roles, metadata, created_at, updated_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Email, &user.Password, &rolesJSON, &metadataJSON, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err := s.unmarshalFields(&user, rolesJSON, metadataJSON); err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates an existing user in Cassandra.
func (s *CassandraStorage) UpdateUser(user *core.User) error {
	// Check if user exists
	if _, err := s.GetUserById(user.ID); err != nil {
		return err
	}

	user.UpdatedAt = time.Now()

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(user.Metadata)
	if err != nil {
		return err
	}

	err = s.session.Query(
		`UPDATE users SET email = ?, password = ?, roles = ?, metadata = ?, updated_at = ? WHERE id = ?`,
		user.Email, user.Password, string(rolesJSON), string(metadataJSON), user.UpdatedAt, user.ID,
	).Exec()
	if err != nil {
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

// DeleteUser removes a user from Cassandra.
func (s *CassandraStorage) DeleteUser(id string) error {
	// Check if user exists
	if _, err := s.GetUserById(id); err != nil {
		return err
	}

	err := s.session.Query(`DELETE FROM users WHERE id = ?`, id).Exec()
	if err != nil {
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

func (s *CassandraStorage) unmarshalFields(user *core.User, rolesJSON, metadataJSON string) error {
	if rolesJSON != "" {
		if err := json.Unmarshal([]byte(rolesJSON), &user.Roles); err != nil {
			return err
		}
	}
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &user.Metadata); err != nil {
			return err
		}
	}
	return nil
}
