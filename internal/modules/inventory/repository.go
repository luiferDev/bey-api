package inventory

import (
	"errors"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type InventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) Create(inventory *Inventory) error {
	return r.db.Create(inventory).Error
}

func (r *InventoryRepository) FindByID(id uuid.UUID) (*Inventory, error) {
	var inventory Inventory
	if err := r.db.First(&inventory, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inventory, nil
}

func (r *InventoryRepository) FindByProductID(productID uuid.UUID) (*Inventory, error) {
	var inventory Inventory
	if err := r.db.Where("product_id = ?", productID).First(&inventory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inventory, nil
}

func (r *InventoryRepository) Update(inventory *Inventory) error {
	return r.db.Save(inventory).Error
}

func (r *InventoryRepository) UpdateQuantity(productID uuid.UUID, quantity int) error {
	return r.db.Model(&Inventory{}).Where("product_id = ?", productID).Update("quantity", quantity).Error
}

func (r *InventoryRepository) Reserve(productID uuid.UUID, quantity int) error {
	return r.db.Model(&Inventory{}).Where("product_id = ?", productID).Updates(map[string]interface{}{
		"quantity": gorm.Expr("quantity - ?", quantity),
		"reserved": gorm.Expr("reserved + ?", quantity),
	}).Error
}

func (r *InventoryRepository) Release(productID uuid.UUID, quantity int) error {
	return r.db.Model(&Inventory{}).Where("product_id = ? AND reserved >= ?", productID, quantity).Updates(map[string]interface{}{
		"quantity": gorm.Expr("quantity + ?", quantity),
		"reserved": gorm.Expr("reserved - ?", quantity),
	}).Error
}

// VariantStockSummary returns the total stock and reserved across all variants of a product.
func (r *InventoryRepository) GetVariantStockSummary(productID uuid.UUID) (totalStock, totalReserved int, err error) {
	type stockResult struct {
		TotalStock    int `gorm:"column:total_stock"`
		TotalReserved int `gorm:"column:total_reserved"`
	}
	var result stockResult
	err = r.db.Raw(`
		SELECT COALESCE(SUM(stock), 0) as total_stock,
		       COALESCE(SUM(reserved), 0) as total_reserved
		FROM product_variants
		WHERE product_id = ? AND deleted_at IS NULL
	`, productID).Scan(&result).Error
	return result.TotalStock, result.TotalReserved, err
}
