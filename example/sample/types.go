package sample

import "time"

// User represents a simple user struct
// @polygon sql/table
// @polygon sql/override name:users
type User struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Email     *string   `json:"email,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Repository interface for user operations
type Repository interface {
	Get(id uint64) (*User, error)
	List() ([]User, error)
	Create(user *User) error
}
