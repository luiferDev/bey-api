package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResponseHandler struct{}

func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{}
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (h *ResponseHandler) Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ApiResponse{
		Success: true,
		Data:    data,
	})
}

func (h *ResponseHandler) Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, ApiResponse{
		Success: true,
		Data:    data,
	})
}

func (h *ResponseHandler) Error(c *gin.Context, status int, message string) {
	c.JSON(status, ApiResponse{
		Success: false,
		Error:   message,
	})
}

func (h *ResponseHandler) ValidationError(c *gin.Context, message string) {
	h.Error(c, http.StatusBadRequest, message)
}

func (h *ResponseHandler) NotFound(c *gin.Context, message string) {
	h.Error(c, http.StatusNotFound, message)
}

func (h *ResponseHandler) InternalError(c *gin.Context, message string) {
	h.Error(c, http.StatusInternalServerError, message)
}
