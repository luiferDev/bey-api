package inventory

import (
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

	inventory, err := h.repo.FindByProductID(productID)
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}
	if inventory == nil {
		h.resp.NotFound(c, "inventory not found")
		return
	}

	resp := toInventoryResponse(inventory)

	// Sum stock from all variants of this product
	variantStock, variantReserved, err := h.repo.GetVariantStockSummary(productID)
	if err == nil && (variantStock > 0 || variantReserved > 0) {
		resp.VariantStock = variantStock
		resp.VariantReserved = variantReserved
		resp.VariantAvailable = variantStock - variantReserved
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

	inventory, err := h.repo.FindByProductID(productID)
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}

	if inventory == nil {
		inventory = &Inventory{
			ProductID: productID,
			Quantity:  0,
			Reserved:  0,
		}
		if err := h.repo.Create(inventory); err != nil {
			h.resp.InternalError(c, "failed to create inventory")
			return
		}
	}

	var req UpdateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if req.Quantity != nil {
		inventory.Quantity = *req.Quantity
	}

	if err := h.repo.Update(inventory); err != nil {
		h.resp.InternalError(c, "failed to update inventory")
		return
	}

	h.resp.Success(c, toInventoryResponse(inventory))
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

	inventory, err := h.repo.FindByProductID(productID)
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}

	if inventory == nil {
		inventory = &Inventory{
			ProductID: productID,
			Quantity:  0,
			Reserved:  0,
		}
		if err := h.repo.Create(inventory); err != nil {
			h.resp.InternalError(c, "failed to create inventory")
			return
		}
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if inventory.Quantity < req.Quantity {
		h.resp.Error(c, 400, "insufficient inventory")
		return
	}

	if err := h.repo.Reserve(productID, req.Quantity); err != nil {
		h.resp.InternalError(c, "failed to reserve inventory")
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

	inventory, err := h.repo.FindByProductID(productID)
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}
	if inventory == nil {
		h.resp.NotFound(c, "inventory not found")
		return
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if inventory.Reserved < req.Quantity {
		h.resp.Error(c, 400, "not enough reserved inventory")
		return
	}

	if err := h.repo.Release(productID, req.Quantity); err != nil {
		h.resp.InternalError(c, "failed to release inventory")
		return
	}

	h.resp.Success(c, gin.H{"message": "inventory released"})
}

func toInventoryResponse(inventory *Inventory) InventoryResponse {
	return InventoryResponse{
		ID:        inventory.ID.String(),
		ProductID: inventory.ProductID.String(),
		Quantity:  inventory.Quantity,
		Reserved:  inventory.Reserved,
		Available: inventory.Quantity - inventory.Reserved,
		UpdatedAt: inventory.UpdatedAt,
	}
}
