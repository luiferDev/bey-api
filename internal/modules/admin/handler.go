package admin

import (
	"bey/internal/modules/users"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db   *gorm.DB
	resp *response.ResponseHandler
	repo *users.UserRepository
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		db:   db,
		resp: response.NewResponseHandler(),
		repo: users.NewUserRepository(db),
	}
}

// CreateUser godoc
// @Summary Create a new user (admin only)
// @Description Creates a new user account. Requires admin role.
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User creation request"
// @Success 201 {object} response.ApiResponse{data=UserResponse} "User created successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid user data or email already exists"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 403 {object} response.ApiResponse "Forbidden - admin role required"
// @Failure 500 {object} response.ApiResponse "Internal server error - user creation failed"
// @Security BearerAuth
// @Router /api/v1/admin/users [post]
func (h *AdminHandler) CreateUser(c *gin.Context) {
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

	creator := users.NewAdminUserCreator(h.db)
	user, err := creator.Create(&users.CreateUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})

	if err != nil {
		h.resp.InternalError(c, "failed to create user")
		return
	}

	h.resp.Created(c, toUserResponse(user))
}

func toUserResponse(user *users.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.FirstName,
		Role:      user.Role,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
	}
}
