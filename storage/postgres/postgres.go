package postgres

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/AryanAg08/loginfy.go/core"
	"github.com/AryanAg08/loginfy.go/pkg/logger"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// PostgresStorage is a PostgreSQL implementation of the Storage interface.
type PostgresStorage struct {
	db  *sql.DB
	log *logger.ServiceLogger
}

// New creates a new PostgreSQL storage instance by opening a connection with the given connection string.
func New(connString string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, err
	}
	s := &PostgresStorage{
		db:  db,
		log: logger.NewServiceLogger("postgres-storage"),
	}
	if err := s.autoMigrate(); err != nil {
		db.Close()
		return nil, err
	}
	s.log.Info("postgres storage initialized", nil)
	return s, nil
}

// NewFromDB creates a PostgresStorage from an existing *sql.DB connection.
func NewFromDB(db *sql.DB) *PostgresStorage {
	s := &PostgresStorage{
		db:  db,
		log: logger.NewServiceLogger("postgres-storage"),
	}
	return s
}

func (s *PostgresStorage) autoMigrate() error {
	query := `CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		roles JSONB DEFAULT '[]',
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
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
func (s *PostgresStorage) Connect() error {
	return s.db.Ping()
}

// Close closes the database connection.
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// Ping checks if the database is reachable.
func (s *PostgresStorage) Ping() error {
	return s.db.Ping()
}

// CreateUser creates a new user in PostgreSQL.
func (s *PostgresStorage) CreateUser(user *core.User) error {
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

	query := `INSERT INTO users (id, email, password, roles, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
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
func (s *PostgresStorage) GetUserByEmail(email string) (*core.User, error) {
	query := `SELECT id, email, password, roles, metadata, created_at, updated_at FROM users WHERE email = $1`
	row := s.db.QueryRow(query, email)
	return s.scanUser(row)
}

// GetUserById retrieves a user by ID.
func (s *PostgresStorage) GetUserById(id string) (*core.User, error) {
	query := `SELECT id, email, password, roles, metadata, created_at, updated_at FROM users WHERE id = $1`
	row := s.db.QueryRow(query, id)
	return s.scanUser(row)
}

// UpdateUser updates an existing user in PostgreSQL.
func (s *PostgresStorage) UpdateUser(user *core.User) error {
	user.UpdatedAt = time.Now()

	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(user.Metadata)
	if err != nil {
		return err
	}

	query := `UPDATE users SET email = $1, password = $2, roles = $3, metadata = $4, updated_at = $5 WHERE id = $6`
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

// DeleteUser removes a user from PostgreSQL.
func (s *PostgresStorage) DeleteUser(id string) error {
	query := `DELETE FROM users WHERE id = $1`
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

func (s *PostgresStorage) scanUser(row *sql.Row) (*core.User, error) {
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
