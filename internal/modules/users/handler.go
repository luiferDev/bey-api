package users

import (
	"golang.org/x/crypto/bcrypt"

	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	repo *UserRepository
	resp *response.ResponseHandler
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{
		repo: NewUserRepository(db),
		resp: response.NewResponseHandler(),
	}
}

// @Summary Create a new user
// @Description Creates a new user account
// @Tags Users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User data"
// @Success 201 {object} UserResponse
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	existing, _ := h.repo.FindByEmail(req.Email)
	if existing != nil {
		h.resp.Error(c, 400, "email already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.resp.InternalError(c, "failed to hash password")
		return
	}

	user := &User{
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      "customer",
		Active:    true,
	}

	if err := h.repo.Create(user); err != nil {
		h.resp.InternalError(c, "failed to create user")
		return
	}

	h.resp.Created(c, toUserResponse(user))
}

// @Summary Get user by ID
// @Description Retrieves a user by their ID
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} UserResponse
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.GetUint("user_id")
	user, err := h.repo.FindByID(id)
	if err != nil {
		h.resp.InternalError(c, "failed to get user")
		return
	}
	if user == nil {
		h.resp.NotFound(c, "user not found")
		return
	}
	h.resp.Success(c, toUserResponse(user))
}

// @Summary Update a user
// @Description Updates an existing user
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body UpdateUserRequest true "User data"
// @Success 200 {object} UserResponse
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id := c.GetUint("user_id")
	user, err := h.repo.FindByID(id)
	if err != nil {
		h.resp.InternalError(c, "failed to get user")
		return
	}
	if user == nil {
		h.resp.NotFound(c, "user not found")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Active != nil {
		user.Active = *req.Active
	}

	if err := h.repo.Update(user); err != nil {
		h.resp.InternalError(c, "failed to update user")
		return
	}

	h.resp.Success(c, toUserResponse(user))
}

// @Summary Delete a user
// @Description Deletes a user by ID
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} gin.H
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.GetUint("user_id")
	if err := h.repo.Delete(id); err != nil {
		h.resp.InternalError(c, "failed to delete user")
		return
	}
	h.resp.Success(c, gin.H{"message": "user deleted"})
}

// @Summary List all users
// @Description Retrieves a list of all users
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {array} UserResponse
// @Router /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.repo.FindAll(0, 100)
	if err != nil {
		h.resp.InternalError(c, "failed to list users")
		return
	}

	responses := make([]UserResponse, len(users))
	for i := range users {
		responses[i] = toUserResponse(&users[i])
	}
	h.resp.Success(c, responses)
}

func toUserResponse(user *User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
	}
}
