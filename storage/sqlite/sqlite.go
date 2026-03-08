package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "modernc.org/sqlite"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// SQLiteStorage is a SQLite implementation of the Storage interface using pure Go (no CGO).
type SQLiteStorage struct {
	db  *sql.DB
	log *logger.ServiceLogger
}

// New creates a new SQLite storage instance by opening or creating a database at dbPath.
func New(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	s := &SQLiteStorage{
		db:  db,
		log: logger.NewServiceLogger("sqlite-storage"),
	}
	if err := s.configure(); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.autoMigrate(); err != nil {
		db.Close()
		return nil, err
	}
	s.log.Info("sqlite storage initialized", map[string]interface{}{
		"path": dbPath,
	})
	return s, nil
}

// NewFromDB creates a SQLiteStorage from an existing *sql.DB connection.
func NewFromDB(db *sql.DB) *SQLiteStorage {
	return &SQLiteStorage{
		db:  db,
		log: logger.NewServiceLogger("sqlite-storage"),
	}
}

func (s *SQLiteStorage) configure() error {
	// Enable WAL mode for better concurrent read performance
	if _, err := s.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return err
	}
	// Enable foreign keys
	if _, err := s.db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStorage) autoMigrate() error {
	query := `CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		roles TEXT DEFAULT '[]',
		metadata TEXT DEFAULT '{}',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`
	_, err := s.db.Exec(query)
	if err != nil {
		s.log.Error("failed to auto-migrate table", map[string]interface{}{
			"error": err.Error(),
		})
	}
	return err
}

// Close closes the database connection.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// CreateUser creates a new user in SQLite.
func (s *SQLiteStorage) CreateUser(user *core.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(user.Metadata)
	if err != nil {
		return err
	}

	query := `INSERT INTO users (id, email, password, roles, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(query, user.ID, user.Email, user.Password, string(rolesJSON), string(metadataJSON), user.CreatedAt, user.UpdatedAt)
	if err != nil {
		s.log.Warn("user creation failed", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return ErrUserAlreadyExists
	}

	s.log.Info("user created successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})
	return nil
}

// GetUserByEmail retrieves a user by email address.
func (s *SQLiteStorage) GetUserByEmail(email string) (*core.User, error) {
	query := `SELECT id, email, password, roles, metadata, created_at, updated_at FROM users WHERE email = ?`
	row := s.db.QueryRow(query, email)
	return s.scanUser(row)
}

// GetUserById retrieves a user by ID.
func (s *SQLiteStorage) GetUserById(id string) (*core.User, error) {
	query := `SELECT id, email, password, roles, metadata, created_at, updated_at FROM users WHERE id = ?`
	row := s.db.QueryRow(query, id)
	return s.scanUser(row)
}

// UpdateUser updates an existing user in SQLite.
func (s *SQLiteStorage) UpdateUser(user *core.User) error {
	user.UpdatedAt = time.Now()

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(user.Metadata)
	if err != nil {
		return err
	}

	query := `UPDATE users SET email = ?, password = ?, roles = ?, metadata = ?, updated_at = ? WHERE id = ?`
	result, err := s.db.Exec(query, user.Email, user.Password, string(rolesJSON), string(metadataJSON), user.UpdatedAt, user.ID)
	if err != nil {
		s.log.Warn("user update failed", map[string]interface{}{
			"user_id": user.ID,
			"error":   err.Error(),
		})
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	s.log.Info("user updated successfully", map[string]interface{}{
		"user_id": user.ID,
	})
	return nil
}

// DeleteUser removes a user from SQLite.
func (s *SQLiteStorage) DeleteUser(id string) error {
	query := `DELETE FROM users WHERE id = ?`
	result, err := s.db.Exec(query, id)
	if err != nil {
		s.log.Warn("user deletion failed", map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		})
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	s.log.Info("user deleted successfully", map[string]interface{}{
		"user_id": id,
	})
	return nil
}

func (s *SQLiteStorage) scanUser(row *sql.Row) (*core.User, error) {
	var user core.User
	var rolesJSON, metadataJSON string

	err := row.Scan(&user.ID, &user.Email, &user.Password, &rolesJSON, &metadataJSON, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if rolesJSON != "" {
		if err := json.Unmarshal([]byte(rolesJSON), &user.Roles); err != nil {
			return nil, err
		}
	}
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &user.Metadata); err != nil {
			return nil, err
		}
	}

	return &user, nil
}
