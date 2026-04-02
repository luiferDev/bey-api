package inventory

import (
	"bey/internal/shared/response"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

// @Summary Get inventory by product ID
// @Description Retrieves inventory for a specific product
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path int true "Product ID"
// @Success 200 {object} InventoryResponse
// @Router /api/v1/inventory/{product_id} [get]
func (h *InventoryHandler) GetByProductID(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("product_id"), 10, 32)
	if err != nil {
		h.resp.ValidationError(c, "invalid product_id")
		return
	}

	inventory, err := h.repo.FindByProductID(uint(productID))
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}
	if inventory == nil {
		h.resp.NotFound(c, "inventory not found")
		return
	}
	h.resp.Success(c, toInventoryResponse(inventory))
}

// @Summary Update inventory
// @Description Updates inventory quantity for a product (creates if not exists)
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path int true "Product ID"
// @Param inventory body UpdateInventoryRequest true "Inventory data"
// @Success 200 {object} InventoryResponse
// @Router /api/v1/inventory/{product_id} [put]
func (h *InventoryHandler) Update(c *gin.Context) {
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		h.resp.Error(c, 403, "admin access required")
		return
	}

	productID, err := strconv.ParseUint(c.Param("product_id"), 10, 32)
	if err != nil {
		h.resp.ValidationError(c, "invalid product_id")
		return
	}

	inventory, err := h.repo.FindByProductID(uint(productID))
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}

	// Auto-create if not exists
	if inventory == nil {
		inventory = &Inventory{
			ProductID: uint(productID),
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

// @Summary Reserve inventory
// @Description Reserves inventory quantity for a product (creates if not exists)
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path int true "Product ID"
// @Param quantity body object true "Quantity to reserve"
// @Success 200
// @Router /api/v1/inventory/{product_id}/reserve [post]
func (h *InventoryHandler) Reserve(c *gin.Context) {
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		h.resp.Error(c, 403, "admin access required")
		return
	}

	productID, err := strconv.ParseUint(c.Param("product_id"), 10, 32)
	if err != nil {
		h.resp.ValidationError(c, "invalid product_id")
		return
	}

	inventory, err := h.repo.FindByProductID(uint(productID))
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}

	// Auto-create if not exists
	if inventory == nil {
		inventory = &Inventory{
			ProductID: uint(productID),
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

	if err := h.repo.Reserve(uint(productID), req.Quantity); err != nil {
		h.resp.InternalError(c, "failed to reserve inventory")
		return
	}

	h.resp.Success(c, gin.H{"message": "inventory reserved"})
}

// @Summary Release inventory
// @Description Releases reserved inventory quantity for a product
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path int true "Product ID"
// @Param quantity body object true "Quantity to release"
// @Success 200
// @Router /api/v1/inventory/{product_id}/release [post]
func (h *InventoryHandler) Release(c *gin.Context) {
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		h.resp.Error(c, 403, "admin access required")
		return
	}

	productID, err := strconv.ParseUint(c.Param("product_id"), 10, 32)
	if err != nil {
		h.resp.ValidationError(c, "invalid product_id")
		return
	}

	inventory, err := h.repo.FindByProductID(uint(productID))
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

	if err := h.repo.Release(uint(productID), req.Quantity); err != nil {
		h.resp.InternalError(c, "failed to release inventory")
		return
	}

	h.resp.Success(c, gin.H{"message": "inventory released"})
}

func toInventoryResponse(inventory *Inventory) InventoryResponse {
	return InventoryResponse{
		ID:        inventory.ID,
		ProductID: inventory.ProductID,
		Quantity:  inventory.Quantity,
		Reserved:  inventory.Reserved,
		Available: inventory.Quantity - inventory.Reserved,
		UpdatedAt: inventory.UpdatedAt,
	}
}
