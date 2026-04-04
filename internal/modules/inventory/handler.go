package inventory

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"

	"bey/internal/shared/response"
)

type InventoryHandler struct {
	repo *InventoryRepository
	resp *response.ResponseHandler
}

func NewInventoryHandler(db *gorm.DB) *InventoryHandler {
	return &InventoryHandler{
		repo: NewInventoryRepository(db),
		resp: response.NewResponseHandler(),
	}
}

func (h *InventoryHandler) GetByProductID(c *gin.Context) {
	productID, err := uuid.FromString(c.Param("product_id"))
	if err != nil {
		h.resp.ValidationError(c, "invalid product_id format")
		return
	}

	result, err := h.repo.GetProductInventory(productID)
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}

	resp := InventoryResponse{
		ProductID:      productID.String(),
		TotalStock:     result.TotalStock,
		TotalReserved:  result.TotalReserved,
		TotalAvailable: result.TotalStock - result.TotalReserved,
		Variants:       result.Variants,
	}

	h.resp.Success(c, resp)
}

func (h *InventoryHandler) Update(c *gin.Context) {
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		h.resp.Error(c, 403, "admin access required")
		return
	}

	productID, err := uuid.FromString(c.Param("product_id"))
	if err != nil {
		h.resp.ValidationError(c, "invalid product_id format")
		return
	}

	var req UpdateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if req.VariantID == nil {
		h.resp.ValidationError(c, "variant_id is required")
		return
	}

	err = h.repo.UpdateVariantStock(*req.VariantID, productID, req.Quantity)
	if err != nil {
		h.resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.repo.GetProductInventory(productID)
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}

	resp := InventoryResponse{
		ProductID:      productID.String(),
		TotalStock:     result.TotalStock,
		TotalReserved:  result.TotalReserved,
		TotalAvailable: result.TotalStock - result.TotalReserved,
		Variants:       result.Variants,
	}

	h.resp.Success(c, resp)
}

func (h *InventoryHandler) Reserve(c *gin.Context) {
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		h.resp.Error(c, 403, "admin access required")
		return
	}

	productID, err := uuid.FromString(c.Param("product_id"))
	if err != nil {
		h.resp.ValidationError(c, "invalid product_id format")
		return
	}

	var req ReserveReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if req.VariantID != nil {
		err = h.repo.ReserveVariantStock(*req.VariantID, req.Quantity)
	} else {
		err = h.repo.ReserveProductStock(productID, req.Quantity)
	}

	if err != nil {
		h.resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.resp.Success(c, gin.H{"message": "inventory reserved"})
}

func (h *InventoryHandler) Release(c *gin.Context) {
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		h.resp.Error(c, 403, "admin access required")
		return
	}

	productID, err := uuid.FromString(c.Param("product_id"))
	if err != nil {
		h.resp.ValidationError(c, "invalid product_id format")
		return
	}

	var req ReserveReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if req.VariantID != nil {
		err = h.repo.ReleaseVariantStock(*req.VariantID, req.Quantity)
	} else {
		err = h.repo.ReleaseProductStock(productID, req.Quantity)
	}

	if err != nil {
		h.resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.resp.Success(c, gin.H{"message": "inventory released"})
}
