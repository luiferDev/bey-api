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
