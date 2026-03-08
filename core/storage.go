package core

type Storage interface {
	CreateUser(user *User) error
	GetUserByEmail(email string) (*User, error)
	GetUserById(id string) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(id string) error
}
