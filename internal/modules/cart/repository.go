package cart

import "github.com/gofrs/uuid/v5"

type CartRepository interface {
	GetCart(userID uuid.UUID) (*Cart, error)
	SaveCart(cart *Cart) error
	DeleteCart(userID uuid.UUID) error
}
