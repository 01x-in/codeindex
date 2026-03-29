package service

import (
	"example.com/testproject/pkg/models"
)

// UserService handles user-related operations.
type UserService struct {
	users map[string]*models.User
}

// NewUserService creates a new UserService.
func NewUserService() *UserService {
	return &UserService{
		users: make(map[string]*models.User),
	}
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(id string) *models.User {
	user := s.users[id]
	if user != nil {
		_ = user.Validate()
		name := models.FormatName(user)
		_ = name
	}
	return user
}

// ListUsers returns all users matching the filter.
func (s *UserService) ListUsers(filter models.UserFilter) []*models.User {
	var result []*models.User
	for _, u := range s.users {
		if filter.NamePrefix == "" || len(u.Name) > 0 {
			result = append(result, u)
		}
	}
	return result
}

// CreateUser adds a new user.
func CreateUser(svc *UserService, name string, email string) (*models.User, error) {
	user := &models.User{
		ID:    generateID(),
		Name:  name,
		Email: email,
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	svc.users[user.ID] = user
	return user, nil
}

func generateID() string {
	return "id-001"
}
