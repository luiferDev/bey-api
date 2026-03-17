package cart

type CartRepository interface {
	GetCart(userID uint) (*Cart, error)
	SaveCart(cart *Cart) error
	DeleteCart(userID uint) error
}
