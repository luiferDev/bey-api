package users

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"

	"bey/internal/shared/response"
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

func (h *UserHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	targetID, err := uuid.FromString(idParam)
	if err != nil {
		h.resp.Error(c, 400, "invalid user id format")
		return
	}

	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" {
		currentUUID, parseErr := uuid.FromString(c.GetString("user_id"))
		if parseErr != nil || currentUUID != targetID {
			h.resp.Error(c, 403, "you can only view your own profile")
			return
		}
		_ = currentUserID
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

func (h *UserHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	targetID, err := uuid.FromString(idParam)
	if err != nil {
		h.resp.Error(c, 400, "invalid user id format")
		return
	}

	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" {
		currentUUID, parseErr := uuid.FromString(c.GetString("user_id"))
		if parseErr != nil || currentUUID != targetID {
			h.resp.Error(c, 403, "you can only update your own profile")
			return
		}
		_ = currentUserID
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

func (h *UserHandler) Delete(c *gin.Context) {
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		h.resp.Error(c, 403, "admin access required")
		return
	}

	idParam := c.Param("id")
	targetID, err := uuid.FromString(idParam)
	if err != nil {
		h.resp.Error(c, 400, "invalid user id format")
		return
	}

	if err := h.repo.Delete(targetID); err != nil {
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

func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	idParam := c.Param("id")
	targetID, err := uuid.FromString(idParam)
	if err != nil {
		h.resp.Error(c, 400, "invalid user id format")
		return
	}

	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" {
		currentUUID, parseErr := uuid.FromString(c.GetString("user_id"))
		if parseErr != nil || currentUUID != targetID {
			h.resp.Error(c, 403, "you can only update your own avatar")
			return
		}
		_ = currentUserID
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
		ID:        user.ID.String(),
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
