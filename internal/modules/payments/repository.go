package payments

import (
	"errors"
	"log"

	"github.com/gofrs/uuid/v5"
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

func (r *PaymentRepository) FindByID(id uuid.UUID) (*Payment, error) {
	var payment Payment
	if err := r.db.First(&payment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment by id %s: %v", id.String(), err)
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

func (r *PaymentRepository) FindByOrderID(orderID uuid.UUID) ([]Payment, error) {
	var payments []Payment
	if err := r.db.Where("order_id = ?", orderID).Find(&payments).Error; err != nil {
		log.Printf("ERROR: Failed to find payments by order id %s: %v", orderID.String(), err)
		return nil, err
	}
	return payments, nil
}

func (r *PaymentRepository) Update(payment *Payment) error {
	if err := r.db.Save(payment).Error; err != nil {
		log.Printf("ERROR: Failed to update payment %s: %v", payment.ID.String(), err)
		return err
	}
	return nil
}

func (r *PaymentRepository) UpdateStatus(id uuid.UUID, status string) error {
	if err := r.db.Model(&Payment{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		log.Printf("ERROR: Failed to update payment status %s: %v", id.String(), err)
		return err
	}
	return nil
}

func (r *PaymentRepository) Delete(id uuid.UUID) error {
	if err := r.db.Delete(&Payment{}, id).Error; err != nil {
		log.Printf("ERROR: Failed to delete payment %s: %v", id.String(), err)
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

func (r *PaymentLinkRepository) FindByID(id uuid.UUID) (*PaymentLink, error) {
	var link PaymentLink
	if err := r.db.First(&link, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment link by id %s: %v", id.String(), err)
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

func (r *PaymentLinkRepository) FindByOrderID(orderID uuid.UUID) (*PaymentLink, error) {
	var link PaymentLink
	if err := r.db.Where("order_id = ?", orderID).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find payment link by order id %s: %v", orderID.String(), err)
		return nil, err
	}
	return &link, nil
}

func (r *PaymentLinkRepository) FindActiveByOrderID(orderID uuid.UUID) (*PaymentLink, error) {
	var link PaymentLink
	if err := r.db.Where("order_id = ? AND status = ?", orderID, StatusActive).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find active payment link by order id %s: %v", orderID.String(), err)
		return nil, err
	}
	return &link, nil
}

func (r *PaymentLinkRepository) Update(link *PaymentLink) error {
	if err := r.db.Save(link).Error; err != nil {
		log.Printf("ERROR: Failed to update payment link %s: %v", link.ID.String(), err)
		return err
	}
	return nil
}

func (r *PaymentLinkRepository) UpdateStatus(id uuid.UUID, status string) error {
	if err := r.db.Model(&PaymentLink{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		log.Printf("ERROR: Failed to update payment link status %s: %v", id.String(), err)
		return err
	}
	return nil
}

func (r *PaymentLinkRepository) Delete(id uuid.UUID) error {
	if err := r.db.Delete(&PaymentLink{}, id).Error; err != nil {
		log.Printf("ERROR: Failed to delete payment link %s: %v", id.String(), err)
		return err
	}
	return nil
}

func (r *PaymentLinkRepository) MarkAsUsed(id uuid.UUID) error {
	if err := r.db.Model(&PaymentLink{}).Where("id = ?", id).Update("status", StatusUsed).Error; err != nil {
		log.Printf("ERROR: Failed to mark payment link as used %s: %v", id.String(), err)
		return err
	}
	return nil
}
