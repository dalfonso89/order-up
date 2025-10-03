package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

var (
	// ErrOrderNotFound is returned when the specified order cannot be found
	ErrOrderNotFound = errors.New("order not found")

	// ErrOrderExists is returned when a new order is being inserted but an order
	// with the same ID already exists
	ErrOrderExists = errors.New("order already exists")
)

////////////////////////////////////////////////////////////////////////////////

// GetOrder should return the order with the given ID. If that ID isn't found then
// the special ErrOrderNotFound error should be returned.
func (i *Instance) GetOrder(ctx context.Context, id string) (Order, error) {
	// TODO: get order from DB based on the id
	var order Order
	var lineItemsJSON string

	query := `SELECT id, customer_email, line_items, status FROM orders WHERE id = ?`

	// Execute the query and scan the results into variables
	err := i.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.CustomerEmail,
		&lineItemsJSON,
		&order.Status,
	)

	// Handle the result
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows found means the order doesn't exist
			return Order{}, ErrOrderNotFound
		}
		// Some other database error occurred
		return Order{}, err
	}

	// Parse the JSON line items back into the LineItems slice
	err = json.Unmarshal([]byte(lineItemsJSON), &order.LineItems)
	if err != nil {
		return Order{}, err
	}

	return order, nil
}

////////////////////////////////////////////////////////////////////////////////

// GetOrders should return all orders with the given status. If status is the
// special -1 value then it should return all orders regardless of their status.
func (i *Instance) GetOrders(ctx context.Context, status OrderStatus) ([]Order, error) {
	var orders []Order

	// Get the rows from the database based on status sent, unless status is -1
	var query string
	if status == -1 {
		query = `SELECT id, customer_email, line_items, status FROM orders`
	} else {
		query = `SELECT id, customer_email, line_items, status FROM orders WHERE status = ?`
	}

	rows, err := i.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Loop through the rows and add the orders to the orders slice
	for rows.Next() {
		var order Order
		var lineItemsJSON string
		err = rows.Scan(
			&order.ID,
			&order.CustomerEmail,
			&lineItemsJSON,
			&order.Status,
		)
		if err != nil {
			return nil, err
		}

		// Parse the JSON line items back into the LineItems slice
		err = json.Unmarshal([]byte(lineItemsJSON), &order.LineItems)
		if err != nil {
			return nil, err
		}

		// Add the order to the orders slice
		orders = append(orders, order)
	}

	return orders, nil
}

////////////////////////////////////////////////////////////////////////////////

// SetOrderStatus should update the order with the given ID and set the status
// field. If that ID isn't found then the special ErrOrderNotFound error should
// be returned.
func (i *Instance) SetOrderStatus(ctx context.Context, id string, status OrderStatus) error {
	// TODO: update the order's status field to status for the id

	// Update the order's status field to status for the id
	query := `UPDATE orders SET status = ? WHERE id = ?`

	result, err := i.db.ExecContext(ctx, query, status, id)

	if err != nil {
		return err
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrOrderNotFound
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

// InsertOrder should fill in the order's ID with a unique identifier if it's not
// already set and then insert it into the database. It should return the order's
// ID. If the order already exists then ErrOrderExists should be returned.
func (i *Instance) InsertOrder(ctx context.Context, order Order) (string, error) {
	// TODO: if the order's ID field is empty, generate a random ID, then insert
	// into the database

	// Generate a random ID if the order's ID field is empty
	if order.ID == "" {
		order.ID = uuid.New().String()
	}

	// Check if order already exists
	_, err := i.GetOrder(ctx, order.ID)
	if err == nil {
		// Order already exists
		return "", ErrOrderExists
	}

	// Insert the order into the database
	query := `INSERT INTO orders (id, customer_email, line_items, status) VALUES (?, ?, ?, ?)`

	orderLineItemsJSON, err := json.Marshal(order.LineItems)
	if err != nil {
		return order.ID, err
	}

	_, err = i.db.ExecContext(ctx, query, order.ID, order.CustomerEmail, orderLineItemsJSON, order.Status)
	if err != nil {
		return order.ID, err
	}

	return order.ID, nil
}
