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

func (h *UserHandler) Delete(c *gin.Context) {
	id := c.GetUint("user_id")
	if err := h.repo.Delete(id); err != nil {
		h.resp.InternalError(c, "failed to delete user")
		return
	}
	h.resp.Success(c, gin.H{"message": "user deleted"})
}

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
