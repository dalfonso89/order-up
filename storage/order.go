package storage

// OrderStatus describes the current status of the order
type OrderStatus int64

const (
	// OrderStatusPending means we haven't charged the customer yet
	OrderStatusPending OrderStatus = 0

	// OrderStatusCharged means we've successfully charged the customer
	OrderStatusCharged OrderStatus = 1

	// OrderStatusFulfilled means we've successfully fulfilled ALL of the line
	// items and we've begun to ship the order
	OrderStatusFulfilled OrderStatus = 2
)

// LineItem is a single charge on an order. The product of the PriceCents and
// Quantity is the total price of the line item.
type LineItem struct {
	// Description is a product ID or a discount ID
	Description string `json:"description"`
	// PriceCents is the individual price that should be multiplied against
	// quantity. For discounts, this value might be less than 0.
	PriceCents int64 `json:"priceCents"`
	// Quantity is how many descriptions this line item represents
	Quantity int64 `json:"quantity"`
}

// Order represents a single order for one or more products
type Order struct {
	// ID is the unique identifier for the order that never changes throughout the
	// order's lifecycle
	ID string `json:"id"`
	// CustomerEmail is the email address of the customer who placed the order
	CustomerEmail string `json:"customerEmail"`
	// LineItems holds the actual products, or discounts, that apply to the order
	LineItems []LineItem `json:"lineItems"`
	// Status represents the current state of the order throughout the
	// pending->charged->fulfilled lifecycle
	Status OrderStatus `json:"status"`
}

// TotalCents is a helper function that loops over each line item and totals up
// the amount to charge for the whole order
func (o Order) TotalCents() int64 {
	var total int64
	for _, li := range o.LineItems {
		total += li.PriceCents * li.Quantity
	}
	return total
}
