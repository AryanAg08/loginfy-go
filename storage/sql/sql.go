package sql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// SQLStorage is a generic SQL implementation of the Storage interface.
// It works with any database/sql compatible driver (postgres, mysql, sqlite).
type SQLStorage struct {
	db        *sql.DB
	tableName string
	log       *logger.ServiceLogger
}

// New creates a new SQL storage instance with the given database connection and table name.
func New(db *sql.DB, tableName string) *SQLStorage {
	if tableName == "" {
		tableName = "users"
	}
	s := &SQLStorage{
		db:        db,
		tableName: tableName,
		log:       logger.NewServiceLogger("sql-storage"),
	}
	return s
}

// AutoMigrate creates the users table if it does not exist.
func (s *SQLStorage) AutoMigrate() error {
	query := `CREATE TABLE IF NOT EXISTS ` + s.tableName + ` (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		roles TEXT,
		metadata TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`
	_, err := s.db.Exec(query)
	if err != nil {
		s.log.Error("failed to create table", map[string]interface{}{
			"table": s.tableName,
			"error": err.Error(),
		})
		return err
	}
	s.log.Info("table auto-migrated", map[string]interface{}{
		"table": s.tableName,
	})
	return nil
}

// CreateUser creates a new user in the database.
func (s *SQLStorage) CreateUser(user *core.User) error {
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

	query := `INSERT INTO ` + s.tableName + ` (id, email, password, roles, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
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
func (s *SQLStorage) GetUserByEmail(email string) (*core.User, error) {
	query := `SELECT id, email, password, roles, metadata, created_at, updated_at FROM ` + s.tableName + ` WHERE email = ?`
	row := s.db.QueryRow(query, email)
	return s.scanUser(row)
}

// GetUserById retrieves a user by ID.
func (s *SQLStorage) GetUserById(id string) (*core.User, error) {
	query := `SELECT id, email, password, roles, metadata, created_at, updated_at FROM ` + s.tableName + ` WHERE id = ?`
	row := s.db.QueryRow(query, id)
	return s.scanUser(row)
}

// UpdateUser updates an existing user.
func (s *SQLStorage) UpdateUser(user *core.User) error {
	user.UpdatedAt = time.Now()

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(user.Metadata)
	if err != nil {
		return err
	}

	query := `UPDATE ` + s.tableName + ` SET email = ?, password = ?, roles = ?, metadata = ?, updated_at = ? WHERE id = ?`
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

// DeleteUser removes a user from the database.
func (s *SQLStorage) DeleteUser(id string) error {
	query := `DELETE FROM ` + s.tableName + ` WHERE id = ?`
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

// Close closes the underlying database connection.
func (s *SQLStorage) Close() error {
	return s.db.Close()
}

func (s *SQLStorage) scanUser(row *sql.Row) (*core.User, error) {
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
