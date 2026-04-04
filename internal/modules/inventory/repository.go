package inventory

import (
	"database/sql"
	"errors"
	"fmt"

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

type ProductInventoryResult struct {
	TotalStock    int
	TotalReserved int
	Variants      []VariantStockInfo
}

func (r *InventoryRepository) GetProductInventory(productID uuid.UUID) (*ProductInventoryResult, error) {
	var totalStock, totalReserved int

	err := r.db.Raw(`
		SELECT COALESCE(SUM(stock), 0), COALESCE(SUM(reserved), 0)
		FROM product_variants
		WHERE product_id = ?
	`, productID).Row().Scan(&totalStock, &totalReserved)
	if err != nil {
		return nil, fmt.Errorf("failed to get product inventory totals: %w", err)
	}

	type variantRow struct {
		ID       string
		SKU      string
		Stock    int
		Reserved int
	}

	var rows []variantRow
	err = r.db.Raw(`
		SELECT id, sku, stock, reserved
		FROM product_variants
		WHERE product_id = ?
		ORDER BY sku
	`, productID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get variant stock details: %w", err)
	}

	variants := make([]VariantStockInfo, 0, len(rows))
	for _, v := range rows {
		variants = append(variants, VariantStockInfo{
			VariantID: v.ID,
			SKU:       v.SKU,
			Stock:     v.Stock,
			Reserved:  v.Reserved,
			Available: v.Stock - v.Reserved,
		})
	}

	return &ProductInventoryResult{
		TotalStock:    totalStock,
		TotalReserved: totalReserved,
		Variants:      variants,
	}, nil
}

func (r *InventoryRepository) ReserveVariantStock(variantID uuid.UUID, quantity int) error {
	result := r.db.Exec(`
		UPDATE product_variants
		SET stock = stock - ?, reserved = reserved + ?
		WHERE id = ? AND stock >= ?
	`, quantity, quantity, variantID, quantity)

	if result.Error != nil {
		return fmt.Errorf("failed to reserve variant stock: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("insufficient stock on variant")
	}
	return nil
}

func (r *InventoryRepository) ReleaseVariantStock(variantID uuid.UUID, quantity int) error {
	result := r.db.Exec(`
		UPDATE product_variants
		SET stock = stock + ?, reserved = reserved - ?
		WHERE id = ? AND reserved >= ?
	`, quantity, quantity, variantID, quantity)

	if result.Error != nil {
		return fmt.Errorf("failed to release variant stock: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("not enough reserved stock on variant")
	}
	return nil
}

func (r *InventoryRepository) ReserveProductStock(productID uuid.UUID, quantity int) error {
	var variantID string
	var stock int

	err := r.db.Raw(`
		SELECT id, stock
		FROM product_variants
		WHERE product_id = ? AND stock >= ?
		ORDER BY stock DESC
		LIMIT 1
	`, productID, quantity).Row().Scan(&variantID, &stock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("insufficient stock across all variants")
		}
		return fmt.Errorf("failed to find variant for reservation: %w", err)
	}

	parsedID, err := uuid.FromString(variantID)
	if err != nil {
		return fmt.Errorf("invalid variant UUID: %w", err)
	}

	return r.ReserveVariantStock(parsedID, quantity)
}

func (r *InventoryRepository) ReleaseProductStock(productID uuid.UUID, quantity int) error {
	var released int
	var variantRows []struct {
		ID       string
		Reserved int
	}

	err := r.db.Raw(`
		SELECT id, reserved
		FROM product_variants
		WHERE product_id = ? AND reserved > 0
		ORDER BY reserved DESC
	`, productID).Scan(&variantRows).Error
	if err != nil {
		return fmt.Errorf("failed to get variants for release: %w", err)
	}

	if len(variantRows) == 0 {
		return errors.New("no reserved stock to release")
	}

	totalReserved := 0
	for _, vr := range variantRows {
		totalReserved += vr.Reserved
	}

	if totalReserved < quantity {
		return errors.New("not enough reserved stock across variants")
	}

	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	remaining := quantity
	for _, vr := range variantRows {
		if remaining <= 0 {
			break
		}

		canRelease := vr.Reserved
		if canRelease > remaining {
			canRelease = remaining
		}

		parsedID, err := uuid.FromString(vr.ID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("invalid variant UUID: %w", err)
		}

		result := tx.Exec(`
			UPDATE product_variants
			SET stock = stock + ?, reserved = reserved - ?
			WHERE id = ?
		`, canRelease, canRelease, parsedID)

		if result.Error != nil {
			tx.Rollback()
			return fmt.Errorf("failed to release stock on variant %s: %w", vr.ID, result.Error)
		}

		released += canRelease
		remaining -= canRelease
	}

	if released < quantity {
		tx.Rollback()
		return errors.New("failed to release full quantity")
	}

	return tx.Commit().Error
}

func (r *InventoryRepository) UpdateVariantStock(variantID uuid.UUID, productID uuid.UUID, quantity int) error {
	var exists int
	err := r.db.Raw(`
		SELECT COUNT(*) FROM product_variants WHERE id = ? AND product_id = ?
	`, variantID, productID).Row().Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check variant existence: %w", err)
	}
	if exists == 0 {
		return errors.New("variant not found for this product")
	}

	result := r.db.Exec(`
		UPDATE product_variants SET stock = ? WHERE id = ?
	`, quantity, variantID)
	if result.Error != nil {
		return fmt.Errorf("failed to update variant stock: %w", result.Error)
	}
	return nil
}
