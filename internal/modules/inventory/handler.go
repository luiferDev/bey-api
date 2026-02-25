package inventory

import (
	"bey/internal/shared/response"

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
	productID := c.GetUint("product_id")
	inventory, err := h.repo.FindByProductID(productID)
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
// @Description Updates inventory quantity for a product
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path int true "Product ID"
// @Param inventory body UpdateInventoryRequest true "Inventory data"
// @Success 200 {object} InventoryResponse
// @Router /api/v1/inventory/{product_id} [put]
func (h *InventoryHandler) Update(c *gin.Context) {
	productID := c.GetUint("product_id")
	inventory, err := h.repo.FindByProductID(productID)
	if err != nil {
		h.resp.InternalError(c, "failed to get inventory")
		return
	}
	if inventory == nil {
		h.resp.NotFound(c, "inventory not found")
		return
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
// @Description Reserves inventory quantity for a product
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path int true "Product ID"
// @Param quantity body object true "Quantity to reserve"
// @Success 200
// @Router /api/v1/inventory/{product_id}/reserve [post]
func (h *InventoryHandler) Reserve(c *gin.Context) {
	productID := c.GetUint("product_id")
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
