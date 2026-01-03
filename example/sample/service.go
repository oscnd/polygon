package sample

import "fmt"

// Service handles user operations
type Service struct {
	repo Repository
}

// NewService creates a new service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ProcessUser processes a user
// @polygon handler
// @polygon auth required:true
func (s *Service) ProcessUser(user *User) error {
	if err := s.repo.Create(user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// Helper function without receiver
func ValidateUser(user *User) bool {
	return user.Name != ""
}
