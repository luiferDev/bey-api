package payments

import (
	"errors"
	"log"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(payment *Payment) error {
	if err := r.db.Create(payment).Error; err != nil {
		log.Printf("ERROR: Failed to create payment: %v", err)
		return err
	}
	return nil
}

func (r *PaymentRepository) FindByID(id uint) (*Payment, error) {
	var payment Payment
	if err := r.db.First(&payment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment by id %d: %v", id, err)
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) FindByWompiTransactionID(wompiID string) (*Payment, error) {
	var payment Payment
	if err := r.db.Where("wompi_transaction_id = ?", wompiID).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment by wompi id %s: %v", wompiID, err)
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) FindByReference(reference string) (*Payment, error) {
	var payment Payment
	if err := r.db.Where("reference = ?", reference).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment by reference %s: %v", reference, err)
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) FindByOrderID(orderID uint) ([]Payment, error) {
	var payments []Payment
	if err := r.db.Where("order_id = ?", orderID).Find(&payments).Error; err != nil {
		log.Printf("ERROR: Failed to find payments by order id %d: %v", orderID, err)
		return nil, err
	}
	return payments, nil
}

func (r *PaymentRepository) Update(payment *Payment) error {
	if err := r.db.Save(payment).Error; err != nil {
		log.Printf("ERROR: Failed to update payment %d: %v", payment.ID, err)
		return err
	}
	return nil
}

func (r *PaymentRepository) UpdateStatus(id uint, status string) error {
	if err := r.db.Model(&Payment{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		log.Printf("ERROR: Failed to update payment status %d: %v", id, err)
		return err
	}
	return nil
}

func (r *PaymentRepository) Delete(id uint) error {
	if err := r.db.Delete(&Payment{}, id).Error; err != nil {
		log.Printf("ERROR: Failed to delete payment %d: %v", id, err)
		return err
	}
	return nil
}

type PaymentLinkRepository struct {
	db *gorm.DB
}

func NewPaymentLinkRepository(db *gorm.DB) *PaymentLinkRepository {
	return &PaymentLinkRepository{db: db}
}

func (r *PaymentLinkRepository) Create(link *PaymentLink) error {
	if err := r.db.Create(link).Error; err != nil {
		log.Printf("ERROR: Failed to create payment link: %v", err)
		return err
	}
	return nil
}

func (r *PaymentLinkRepository) FindByID(id uint) (*PaymentLink, error) {
	var link PaymentLink
	if err := r.db.First(&link, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment link by id %d: %v", id, err)
		return nil, err
	}
	return &link, nil
}

func (r *PaymentLinkRepository) FindByWompiLinkID(wompiID string) (*PaymentLink, error) {
	var link PaymentLink
	if err := r.db.Where("wompi_link_id = ?", wompiID).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment link by wompi id %s: %v", wompiID, err)
		return nil, err
	}
	return &link, nil
}

func (r *PaymentLinkRepository) FindByReference(reference string) (*PaymentLink, error) {
	var link PaymentLink
	if err := r.db.Where("reference = ?", reference).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment link by reference %s: %v", reference, err)
		return nil, err
	}
	return &link, nil
}

func (r *PaymentLinkRepository) FindByOrderID(orderID uint) (*PaymentLink, error) {
	var link PaymentLink
	if err := r.db.Where("order_id = ?", orderID).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment link by order id %d: %v", orderID, err)
		return nil, err
	}
	return &link, nil
}

func (r *PaymentLinkRepository) FindActiveByOrderID(orderID uint) (*PaymentLink, error) {
	var link PaymentLink
	if err := r.db.Where("order_id = ? AND status = ?", orderID, StatusActive).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find active payment link by order id %d: %v", orderID, err)
		return nil, err
	}
	return &link, nil
}

func (r *PaymentLinkRepository) Update(link *PaymentLink) error {
	if err := r.db.Save(link).Error; err != nil {
		log.Printf("ERROR: Failed to update payment link %d: %v", link.ID, err)
		return err
	}
	return nil
}

func (r *PaymentLinkRepository) UpdateStatus(id uint, status string) error {
	if err := r.db.Model(&PaymentLink{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		log.Printf("ERROR: Failed to update payment link status %d: %v", id, err)
		return err
	}
	return nil
}

func (r *PaymentLinkRepository) Delete(id uint) error {
	if err := r.db.Delete(&PaymentLink{}, id).Error; err != nil {
		log.Printf("ERROR: Failed to delete payment link %d: %v", id, err)
		return err
	}
	return nil
}

func (r *PaymentLinkRepository) MarkAsUsed(id uint) error {
	if err := r.db.Model(&PaymentLink{}).Where("id = ?", id).Update("status", StatusUsed).Error; err != nil {
		log.Printf("ERROR: Failed to mark payment link as used %d: %v", id, err)
		return err
	}
	return nil
}
