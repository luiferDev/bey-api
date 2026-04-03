package payments

import (
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPaymentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	db.AutoMigrate(&Payment{}, &PaymentLink{})

	return db
}

func TestPaymentRepository_Create(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)

	tests := []struct {
		name    string
		payment *Payment
		wantErr bool
	}{
		{
			name: "valid payment",
			payment: &Payment{
				OrderID:            uuid.Must(uuid.NewV7()),
				WompiTransactionID: "tx_123",
				Amount:             100000,
				Currency:           "COP",
				Status:             StatusPending,
				Reference:          "order-001",
			},
			wantErr: false,
		},
		{
			name: "payment with minimal fields",
			payment: &Payment{
				Amount:    50000,
				Currency:  "COP",
				Reference: "order-002",
			},
			wantErr: false,
		},
		{
			name: "payment with payment method",
			payment: &Payment{
				OrderID:            uuid.Must(uuid.NewV7()),
				WompiTransactionID: "tx_456",
				Amount:             150000,
				Currency:           "COP",
				Status:             StatusApproved,
				PaymentMethod:      "CARD",
				Reference:          "order-003",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.payment)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.payment.ID == uuid.Nil {
				t.Error("Expected payment ID to be set")
			}
		})
	}
}

func TestPaymentRepository_FindByID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)

	existingPayment := &Payment{
		OrderID:            uuid.Must(uuid.NewV7()),
		WompiTransactionID: "tx_123",
		Amount:             100000,
		Currency:           "COP",
		Status:             StatusPending,
		Reference:          "order-001",
	}
	repo.Create(existingPayment)

	tests := []struct {
		name      string
		id        uuid.UUID
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "found existing payment",
			id:        existingPayment.ID,
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "payment not found",
			id:        uuid.Must(uuid.NewV7()),
			wantErr:   false,
			expectNil: true,
		},
		{
			name:      "zero id",
			id:        uuid.Nil,
			wantErr:   false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := repo.FindByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectNil && payment != nil {
				t.Error("Expected nil payment")
			}
			if !tt.expectNil && payment == nil {
				t.Error("Expected non-nil payment")
			}
			if !tt.expectNil && payment != nil && payment.Amount != 100000 {
				t.Errorf("Expected amount 100000, got %d", payment.Amount)
			}
		})
	}
}

func TestPaymentRepository_FindByOrderID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)

	orderID1 := uuid.Must(uuid.NewV7())
	orderID2 := uuid.Must(uuid.NewV7())
	orderID3 := uuid.Must(uuid.NewV7())

	repo.Create(&Payment{OrderID: orderID1, WompiTransactionID: "tx_001", Amount: 100000, Currency: "COP", Reference: "order-001"})
	repo.Create(&Payment{OrderID: orderID1, WompiTransactionID: "tx_002", Amount: 50000, Currency: "COP", Reference: "order-002"})
	repo.Create(&Payment{OrderID: orderID2, WompiTransactionID: "tx_003", Amount: 75000, Currency: "COP", Reference: "order-003"})

	tests := []struct {
		name      string
		orderID   uuid.UUID
		wantCount int
		wantErr   bool
	}{
		{
			name:      "find payments for order 1",
			orderID:   orderID1,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "find payments for order 2",
			orderID:   orderID2,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "no payments for order",
			orderID:   orderID3,
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payments, err := repo.FindByOrderID(tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByOrderID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(payments) != tt.wantCount {
				t.Errorf("Expected %d payments, got %d", tt.wantCount, len(payments))
			}
		})
	}
}

func TestPaymentRepository_Update(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)

	payment := &Payment{
		OrderID:            uuid.Must(uuid.NewV7()),
		WompiTransactionID: "tx_123",
		Amount:             100000,
		Currency:           "COP",
		Status:             StatusPending,
		Reference:          "order-001",
	}
	repo.Create(payment)

	tests := []struct {
		name    string
		update  func(p *Payment)
		wantErr bool
	}{
		{
			name: "update status",
			update: func(p *Payment) {
				p.Status = StatusApproved
			},
			wantErr: false,
		},
		{
			name: "update amount",
			update: func(p *Payment) {
				p.Amount = 150000
			},
			wantErr: false,
		},
		{
			name: "update payment method",
			update: func(p *Payment) {
				p.PaymentMethod = "CARD"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.update(payment)
			err := repo.Update(payment)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			updated, _ := repo.FindByID(payment.ID)
			if tt.name == "update status" && updated.Status != StatusApproved {
				t.Errorf("Expected status %s, got %s", StatusApproved, updated.Status)
			}
			if tt.name == "update amount" && updated.Amount != 150000 {
				t.Errorf("Expected amount 150000, got %d", updated.Amount)
			}
			if tt.name == "update payment method" && updated.PaymentMethod != "CARD" {
				t.Errorf("Expected payment method CARD, got %s", updated.PaymentMethod)
			}
		})
	}
}

func TestPaymentRepository_Delete(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)

	payment := &Payment{
		OrderID:  uuid.Must(uuid.NewV7()),
		Amount:   100000,
		Currency: "COP",
	}
	repo.Create(payment)
	paymentID := payment.ID

	err := repo.Delete(paymentID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	deleted, _ := repo.FindByID(paymentID)
	if deleted != nil {
		t.Error("Expected payment to be nil after delete")
	}
}

func TestPaymentRepository_FindByWompiTransactionID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)

	existingPayment := &Payment{
		WompiTransactionID: "tx_wompi_123",
		Amount:             100000,
		Currency:           "COP",
		Reference:          "order-001",
	}
	repo.Create(existingPayment)

	tests := []struct {
		name      string
		wompiID   string
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "found by wompi transaction id",
			wompiID:   "tx_wompi_123",
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "not found",
			wompiID:   "tx_nonexistent",
			wantErr:   false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := repo.FindByWompiTransactionID(tt.wompiID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByWompiTransactionID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectNil && payment != nil {
				t.Error("Expected nil payment")
			}
			if !tt.expectNil && payment == nil {
				t.Error("Expected non-nil payment")
			}
		})
	}
}

func TestPaymentRepository_FindByReference(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)

	existingPayment := &Payment{
		Amount:    100000,
		Currency:  "COP",
		Reference: "ref_12345",
	}
	repo.Create(existingPayment)

	tests := []struct {
		name      string
		reference string
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "found by reference",
			reference: "ref_12345",
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "not found",
			reference: "ref_nonexistent",
			wantErr:   false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := repo.FindByReference(tt.reference)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByReference() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectNil && payment != nil {
				t.Error("Expected nil payment")
			}
			if !tt.expectNil && payment == nil {
				t.Error("Expected non-nil payment")
			}
		})
	}
}

func TestPaymentRepository_UpdateStatus(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentRepository(db)

	payment := &Payment{
		Amount:   100000,
		Currency: "COP",
		Status:   StatusPending,
	}
	repo.Create(payment)

	err := repo.UpdateStatus(payment.ID, StatusApproved)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	updated, _ := repo.FindByID(payment.ID)
	if updated.Status != StatusApproved {
		t.Errorf("Expected status %s, got %s", StatusApproved, updated.Status)
	}
}

func TestPaymentRepository_ErrorHandling(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	repo := NewPaymentRepository(db)

	_, err = repo.FindByID(uuid.Must(uuid.NewV7()))
	if err == nil {
		t.Error("Expected error when finding in non-migrated table")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		t.Logf("Got error: %v", err)
	}
}

// ==================== PaymentLink Repository Tests ====================

func TestPaymentLinkRepository_Create(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	tests := []struct {
		name    string
		link    *PaymentLink
		wantErr bool
	}{
		{
			name: "valid payment link",
			link: &PaymentLink{
				OrderID:     uuid.Must(uuid.NewV7()),
				WompiLinkID: "link_123",
				URL:         "https://wompi.co/pay/link_123",
				Amount:      100000,
				Currency:    "COP",
				Description: "Payment for order 001",
				Status:      StatusActive,
			},
			wantErr: false,
		},
		{
			name: "payment link with single use",
			link: &PaymentLink{
				OrderID:     uuid.Must(uuid.NewV7()),
				WompiLinkID: "link_456",
				Amount:      50000,
				Currency:    "COP",
				Status:      StatusActive,
				SingleUse:   true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.link)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.link.ID == uuid.Nil {
				t.Error("Expected link ID to be set")
			}
		})
	}
}

func TestPaymentLinkRepository_FindByID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	existingLink := &PaymentLink{
		OrderID:     uuid.Must(uuid.NewV7()),
		WompiLinkID: "link_123",
		Amount:      100000,
		Currency:    "COP",
		Status:      StatusActive,
	}
	repo.Create(existingLink)

	tests := []struct {
		name      string
		id        uuid.UUID
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "found existing link",
			id:        existingLink.ID,
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "link not found",
			id:        uuid.Must(uuid.NewV7()),
			wantErr:   false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, err := repo.FindByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectNil && link != nil {
				t.Error("Expected nil link")
			}
			if !tt.expectNil && link == nil {
				t.Error("Expected non-nil link")
			}
		})
	}
}

func TestPaymentLinkRepository_FindByOrderID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	orderID1 := uuid.Must(uuid.NewV7())
	orderID2 := uuid.Must(uuid.NewV7())

	repo.Create(&PaymentLink{OrderID: orderID1, WompiLinkID: "link_1", Amount: 100000, Status: StatusActive})
	repo.Create(&PaymentLink{OrderID: orderID2, WompiLinkID: "link_2", Amount: 50000, Status: StatusActive})

	tests := []struct {
		name      string
		orderID   uuid.UUID
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "found link for order",
			orderID:   orderID1,
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "no link for order",
			orderID:   uuid.Must(uuid.NewV7()),
			wantErr:   false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, err := repo.FindByOrderID(tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByOrderID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectNil && link != nil {
				t.Error("Expected nil link")
			}
			if !tt.expectNil && link == nil {
				t.Error("Expected non-nil link")
			}
		})
	}
}

func TestPaymentLinkRepository_FindByWompiLinkID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	existingLink := &PaymentLink{
		WompiLinkID: "wompi_link_123",
		Amount:      100000,
		Currency:    "COP",
		Status:      StatusActive,
	}
	repo.Create(existingLink)

	tests := []struct {
		name      string
		wompiID   string
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "found by wompi link id",
			wompiID:   "wompi_link_123",
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "not found",
			wompiID:   "wompi_link_nonexistent",
			wantErr:   false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, err := repo.FindByWompiLinkID(tt.wompiID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByWompiLinkID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectNil && link != nil {
				t.Error("Expected nil link")
			}
			if !tt.expectNil && link == nil {
				t.Error("Expected non-nil link")
			}
		})
	}
}

func TestPaymentLinkRepository_FindByReference(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	existingLink := &PaymentLink{
		Amount:    100000,
		Currency:  "COP",
		Reference: "payment_ref_123",
		Status:    StatusActive,
	}
	repo.Create(existingLink)

	tests := []struct {
		name      string
		reference string
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "found by reference",
			reference: "payment_ref_123",
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "not found",
			reference: "nonexistent_ref",
			wantErr:   false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, err := repo.FindByReference(tt.reference)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByReference() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectNil && link != nil {
				t.Error("Expected nil link")
			}
			if !tt.expectNil && link == nil {
				t.Error("Expected non-nil link")
			}
		})
	}
}

func TestPaymentLinkRepository_Update(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	link := &PaymentLink{
		OrderID:     uuid.Must(uuid.NewV7()),
		WompiLinkID: "link_123",
		Amount:      100000,
		Currency:    "COP",
		Status:      StatusActive,
	}
	repo.Create(link)

	tests := []struct {
		name    string
		update  func(l *PaymentLink)
		wantErr bool
	}{
		{
			name: "update status to inactive",
			update: func(l *PaymentLink) {
				l.Status = StatusInactive
			},
			wantErr: false,
		},
		{
			name: "update amount",
			update: func(l *PaymentLink) {
				l.Amount = 150000
			},
			wantErr: false,
		},
		{
			name: "update description",
			update: func(l *PaymentLink) {
				l.Description = "Updated description"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.update(link)
			err := repo.Update(link)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPaymentLinkRepository_Delete(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	link := &PaymentLink{
		OrderID:  uuid.Must(uuid.NewV7()),
		Amount:   100000,
		Currency: "COP",
	}
	repo.Create(link)
	linkID := link.ID

	err := repo.Delete(linkID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	deleted, _ := repo.FindByID(linkID)
	if deleted != nil {
		t.Error("Expected link to be nil after delete")
	}
}

func TestPaymentLinkRepository_UpdateStatus(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	link := &PaymentLink{
		Amount:   100000,
		Currency: "COP",
		Status:   StatusActive,
	}
	repo.Create(link)

	err := repo.UpdateStatus(link.ID, StatusUsed)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	updated, _ := repo.FindByID(link.ID)
	if updated.Status != StatusUsed {
		t.Errorf("Expected status %s, got %s", StatusUsed, updated.Status)
	}
}

func TestPaymentLinkRepository_FindActiveByOrderID(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	orderID1 := uuid.Must(uuid.NewV7())
	orderID2 := uuid.Must(uuid.NewV7())
	orderID3 := uuid.Must(uuid.NewV7())

	repo.Create(&PaymentLink{OrderID: orderID1, WompiLinkID: "link_1", Amount: 100000, Status: StatusActive})
	repo.Create(&PaymentLink{OrderID: orderID2, WompiLinkID: "link_2", Amount: 50000, Status: StatusUsed})
	repo.Create(&PaymentLink{OrderID: orderID3, WompiLinkID: "link_3", Amount: 75000, Status: StatusActive})

	tests := []struct {
		name      string
		orderID   uuid.UUID
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "find active link for order",
			orderID:   orderID1,
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "no active link for order",
			orderID:   orderID2,
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "no link at all",
			orderID:   uuid.Must(uuid.NewV7()),
			wantErr:   false,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, err := repo.FindActiveByOrderID(tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindActiveByOrderID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectNil && link != nil {
				t.Error("Expected nil link")
			}
			if !tt.expectNil && link == nil {
				t.Error("Expected non-nil link")
			}
			if !tt.expectNil && link != nil && link.Status != StatusActive {
				t.Errorf("Expected active link, got %s", link.Status)
			}
		})
	}
}

func TestPaymentLinkRepository_MarkAsUsed(t *testing.T) {
	db := setupPaymentTestDB(t)
	repo := NewPaymentLinkRepository(db)

	link := &PaymentLink{
		Amount:    100000,
		Currency:  "COP",
		Status:    StatusActive,
		SingleUse: true,
	}
	repo.Create(link)

	err := repo.MarkAsUsed(link.ID)
	if err != nil {
		t.Fatalf("MarkAsUsed() error = %v", err)
	}

	updated, _ := repo.FindByID(link.ID)
	if updated.Status != StatusUsed {
		t.Errorf("Expected status %s, got %s", StatusUsed, updated.Status)
	}
}

func TestPaymentLinkRepository_ErrorHandling(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	repo := NewPaymentLinkRepository(db)

	_, err = repo.FindByID(uuid.Must(uuid.NewV7()))
	if err == nil {
		t.Error("Expected error when finding in non-migrated table")
	}
}
