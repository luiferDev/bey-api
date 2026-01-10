package inventory

import (
	"errors"

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

func (r *InventoryRepository) FindByID(id uint) (*Inventory, error) {
	var inventory Inventory
	if err := r.db.First(&inventory, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inventory, nil
}

func (r *InventoryRepository) FindByProductID(productID uint) (*Inventory, error) {
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

func (r *InventoryRepository) UpdateQuantity(productID uint, quantity int) error {
	return r.db.Model(&Inventory{}).Where("product_id = ?", productID).Update("quantity", quantity).Error
}

func (r *InventoryRepository) Reserve(productID uint, quantity int) error {
	return r.db.Model(&Inventory{}).Where("product_id = ?", productID).Updates(map[string]interface{}{
		"quantity": gorm.Expr("quantity - ?", quantity),
		"reserved": gorm.Expr("reserved + ?", quantity),
	}).Error
}
