package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// MemoryInstance is an in-memory implementation of the StorageInstance interface.
type MemoryInstance struct {
	m      sync.RWMutex
	orders map[string]Order
}

// NewMemory returns a new in-memory storage instance.
func NewMemory() *MemoryInstance {
	return &MemoryInstance{
		orders: make(map[string]Order),
	}
}

// GetOrder retrieves an order by its ID.
func (i *MemoryInstance) GetOrder(ctx context.Context, id string) (Order, error) {
	i.m.RLock()
	defer i.m.RUnlock()

	order, ok := i.orders[id]
	if !ok {
		return Order{}, ErrOrderNotFound
	}
	return order, nil
}

// GetOrders retrieves all orders, optionally filtered by status.
func (i *MemoryInstance) GetOrders(ctx context.Context, status OrderStatus) ([]Order, error) {
	i.m.RLock()
	defer i.m.RUnlock()

	var orders []Order
	for _, order := range i.orders {
		if status == -1 || order.Status == status {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

// SetOrderStatus updates the status of an order.
func (i *MemoryInstance) SetOrderStatus(ctx context.Context, id string, status OrderStatus) error {
	i.m.Lock()
	defer i.m.Unlock()

	order, ok := i.orders[id]
	if !ok {
		return ErrOrderNotFound
	}
	order.Status = status
	i.orders[id] = order
	return nil
}

// InsertOrder adds a new order to the store.
func (i *MemoryInstance) InsertOrder(ctx context.Context, order Order) (string, error) {
	i.m.Lock()
	defer i.m.Unlock()

	if order.ID == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		order.ID = hex.EncodeToString(b)
	}

	if _, ok := i.orders[order.ID]; ok {
		return "", ErrOrderExists
	}

	i.orders[order.ID] = order
	return order.ID, nil
}
