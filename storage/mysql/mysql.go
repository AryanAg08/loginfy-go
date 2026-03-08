package mysql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// MySQLStorage is a MySQL implementation of the Storage interface.
type MySQLStorage struct {
	db  *sql.DB
	log *logger.ServiceLogger
}

// New creates a new MySQL storage instance by opening a connection with the given DSN.
func New(dsn string) (*MySQLStorage, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	s := &MySQLStorage{
		db:  db,
		log: logger.NewServiceLogger("mysql-storage"),
	}
	if err := s.autoMigrate(); err != nil {
		db.Close()
		return nil, err
	}
	s.log.Info("mysql storage initialized", nil)
	return s, nil
}

// NewFromDB creates a MySQLStorage from an existing *sql.DB connection.
func NewFromDB(db *sql.DB) *MySQLStorage {
	return &MySQLStorage{
		db:  db,
		log: logger.NewServiceLogger("mysql-storage"),
	}
}

func (s *MySQLStorage) autoMigrate() error {
	query := `CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(255) PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		password TEXT NOT NULL,
		roles JSON,
		metadata JSON,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	)`
	_, err := s.db.Exec(query)
	if err != nil {
		s.log.Error("failed to auto-migrate table", map[string]interface{}{
			"error": err.Error(),
		})
	}
	return err
}

// Connect verifies the database connection is alive.
func (s *MySQLStorage) Connect() error {
	return s.db.Ping()
}

// Close closes the database connection.
func (s *MySQLStorage) Close() error {
	return s.db.Close()
}

// Ping checks if the database is reachable.
func (s *MySQLStorage) Ping() error {
	return s.db.Ping()
}

// CreateUser creates a new user in MySQL.
func (s *MySQLStorage) CreateUser(user *core.User) error {
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
func (s *MySQLStorage) GetUserByEmail(email string) (*core.User, error) {
	query := `SELECT id, email, password, roles, metadata, created_at, updated_at FROM users WHERE email = ?`
	row := s.db.QueryRow(query, email)
	return s.scanUser(row)
}

// GetUserById retrieves a user by ID.
func (s *MySQLStorage) GetUserById(id string) (*core.User, error) {
	query := `SELECT id, email, password, roles, metadata, created_at, updated_at FROM users WHERE id = ?`
	row := s.db.QueryRow(query, id)
	return s.scanUser(row)
}

// UpdateUser updates an existing user in MySQL.
func (s *MySQLStorage) UpdateUser(user *core.User) error {
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

// DeleteUser removes a user from MySQL.
func (s *MySQLStorage) DeleteUser(id string) error {
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

func (s *MySQLStorage) scanUser(row *sql.Row) (*core.User, error) {
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
