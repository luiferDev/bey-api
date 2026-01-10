package orders

import (
	"errors"

	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(order *Order) error {
	return r.db.Create(order).Error
}

func (r *OrderRepository) FindByID(id uint) (*Order, error) {
	var order Order
	if err := r.db.Preload("Items").First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) Update(order *Order) error {
	return r.db.Save(order).Error
}

func (r *OrderRepository) Delete(id uint) error {
	return r.db.Delete(&Order{}, id).Error
}

func (r *OrderRepository) FindByUserID(userID uint) ([]Order, error) {
	var orders []Order
	if err := r.db.Preload("Items").Where("user_id = ?", userID).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *OrderRepository) FindAll(offset, limit int) ([]Order, error) {
	var orders []Order
	if err := r.db.Preload("Items").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}
