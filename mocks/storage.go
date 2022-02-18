//

package mocks

import (
	"context"

	"github.com/levenlabs/order-up/storage"
)

// StorageInstance allows us to mock *storage.Instance in the api package
type StorageInstance interface {
	// GetOrder should return the order with the given ID. If that ID isn't found then
	// the special ErrOrderNotFound error should be returned.
	GetOrder(ctx context.Context, id string) (storage.Order, error)
	// GetOrders should return all orders with the given status. If status is the
	// special -1 value then it should return all orders regardless of their status.
	GetOrders(ctx context.Context, status storage.OrderStatus) ([]storage.Order, error)
	// SetOrderStatus should update the order with the given ID and set the status
	// field. If that ID isn't found then the special ErrOrderNotFound error should
	// be returned.
	SetOrderStatus(ctx context.Context, id string, status storage.OrderStatus) error
	// InsertOrder should fill in the order's ID with a unique identifier if it's not
	// already set and then insert it into the database. It should return the order's
	// ID. If the order already exists then ErrOrderExists should be returned.
	InsertOrder(ctx context.Context, order storage.Order) (string, error)
}
