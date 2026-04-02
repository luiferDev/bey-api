package users

import (
	"errors"
	"fmt"
	"unicode"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
	Surname  string `json:"surname"`
	Phone    string `json:"phone"`
}

func validatePasswordComplexity(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, c := range password {
		if unicode.IsUpper(c) {
			hasUpper = true
		}
		if unicode.IsLower(c) {
			hasLower = true
		}
		if unicode.IsDigit(c) {
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("password must contain uppercase, lowercase, and digit")
	}
	return nil
}

type UserCreator interface {
	Create(req *CreateUserRequest) (*User, error)
	buildUser(req *CreateUserRequest, hashedPassword string) *User
}

type RegularUserCreator struct {
	db *gorm.DB
}

func NewRegularUserCreator(db *gorm.DB) *RegularUserCreator {
	return &RegularUserCreator{db: db}
}

func (c *RegularUserCreator) Create(req *CreateUserRequest) (*User, error) {
	if req.Email == "" {
		return nil, errors.New("email is required")
	}

	if err := validatePasswordComplexity(req.Password); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := c.buildUser(req, string(hashedPassword))

	if err := c.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (c *RegularUserCreator) buildUser(req *CreateUserRequest, hashedPassword string) *User {
	return &User{
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.Name,
		LastName:  req.Surname,
		Phone:     req.Phone,
		Role:      "customer",
		Active:    true,
	}
}

type AdminUserCreator struct {
	db *gorm.DB
}

func NewAdminUserCreator(db *gorm.DB) *AdminUserCreator {
	return &AdminUserCreator{db: db}
}

func (c *AdminUserCreator) Create(req *CreateUserRequest) (*User, error) {
	if req.Email == "" {
		return nil, errors.New("email is required")
	}

	if err := validatePasswordComplexity(req.Password); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := c.buildUser(req, string(hashedPassword))

	if err := c.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (c *AdminUserCreator) buildUser(req *CreateUserRequest, hashedPassword string) *User {
	return &User{
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.Name,
		LastName:  req.Surname,
		Phone:     req.Phone,
		Role:      "admin",
		Active:    true,
	}
}
