package users

import (
	"fmt"

	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	repo    *UserRepository
	resp    *response.ResponseHandler
	creator UserCreator
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{
		repo:    NewUserRepository(db),
		resp:    response.NewResponseHandler(),
		creator: NewRegularUserCreator(db),
	}
}

// @Summary Register a new user
// @Description Creates a new user account (public endpoint)
// @Tags Users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User data (name, email, password, surname, phone)"
// @Success 201 {object} UserResponse
// @Router /api/v1/users/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	existing, err := h.repo.FindByEmail(req.Email)
	if err != nil {
		h.resp.InternalError(c, "failed to check existing user")
		return
	}
	if existing != nil {
		h.resp.Error(c, 400, "email already exists")
		return
	}

	user, err := h.creator.Create(&req)
	if err != nil {
		h.resp.InternalError(c, "failed to create user")
		return
	}

	h.resp.Created(c, toUserResponse(user))
}

// @Summary Register a new admin user
// @Description Creates a new admin user account (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "Admin user data"
// @Success 201 {object} UserResponse
// @Router /api/v1/users/register-admin [post]
func (h *UserHandler) RegisterAdmin(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	existing, err := h.repo.FindByEmail(req.Email)
	if err != nil {
		h.resp.InternalError(c, "failed to check existing user")
		return
	}
	if existing != nil {
		h.resp.Error(c, 400, "email already exists")
		return
	}

	adminCreator := NewAdminUserCreator(h.repo.db)
	user, err := adminCreator.Create(&req)
	if err != nil {
		h.resp.InternalError(c, "failed to create admin user")
		return
	}

	h.resp.Created(c, toUserResponse(user))
}

// @Summary Get user by ID
// @Description Retrieves a user by their ID (user themselves or admin)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} UserResponse
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	// Get ID from path parameter
	idParam := c.Param("id")
	var targetID uint
	if _, err := fmt.Sscanf(idParam, "%d", &targetID); err != nil {
		h.resp.Error(c, 400, "invalid user id")
		return
	}

	// Get current user info from token
	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	// Check if current user is admin or is the same user
	if userRole != "admin" && currentUserID != targetID {
		h.resp.Error(c, 403, "you can only view your own profile")
		return
	}

	user, err := h.repo.FindByID(targetID)
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
// @Description Updates an existing user (user themselves or admin)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body UpdateUserRequest true "User data"
// @Success 200 {object} UserResponse
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	// Get ID from path parameter
	idParam := c.Param("id")
	var targetID uint
	if _, err := fmt.Sscanf(idParam, "%d", &targetID); err != nil {
		h.resp.Error(c, 400, "invalid user id")
		return
	}

	// Get current user info from token
	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	// Check if current user is admin or is the same user
	if userRole != "admin" && currentUserID != targetID {
		h.resp.Error(c, 403, "you can only update your own profile")
		return
	}

	user, err := h.repo.FindByID(targetID)
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
// @Description Deletes a user by ID (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} gin.H
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	// Defense-in-depth: explicit admin role check
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		h.resp.Error(c, 403, "admin access required")
		return
	}

	// Get ID from path parameter
	idParam := c.Param("id")
	var targetID uint
	if _, err := fmt.Sscanf(idParam, "%d", &targetID); err != nil {
		h.resp.Error(c, 400, "invalid user id")
		return
	}

	if err := h.repo.Delete(targetID); err != nil {
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

// @Summary Update user avatar
// @Description Updates the avatar URL for a user (user themselves or admin)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param avatar body object true "Avatar URL"
// @Success 200 {object} UserResponse
// @Router /api/v1/users/{id}/avatar [put]
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	// Get ID from path parameter
	idParam := c.Param("id")
	var targetID uint
	if _, err := fmt.Sscanf(idParam, "%d", &targetID); err != nil {
		h.resp.Error(c, 400, "invalid user id")
		return
	}

	// Get current user info from token
	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	// Check if current user is admin or is the same user
	if userRole != "admin" && currentUserID != targetID {
		h.resp.Error(c, 403, "you can only update your own avatar")
		return
	}

	user, err := h.repo.FindByID(targetID)
	if err != nil {
		h.resp.InternalError(c, "failed to get user")
		return
	}
	if user == nil {
		h.resp.NotFound(c, "user not found")
		return
	}

	var req struct {
		AvatarURL string `json:"avatar_url" binding:"required,url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, "avatar_url is required and must be a valid URL")
		return
	}

	user.AvatarURL = req.AvatarURL

	if err := h.repo.Update(user); err != nil {
		h.resp.InternalError(c, "failed to update avatar")
		return
	}

	h.resp.Success(c, toUserResponse(user))
}

func toUserResponse(user *User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		Role:      user.Role,
		Active:    user.Active,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
	}
}
