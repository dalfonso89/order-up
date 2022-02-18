package storage

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomDatabase() string {
	// make a backing array with length 12 and a slice with length 12 as well
	b := make([]byte, 12)
	// rand.Read will read up to len(b) with random bytes, in this case 12
	_, err := rand.Read(b)
	if err != nil {
		// rand.Read should never error unless we run out of entropy and since this
		// is just in tests anyways it's easier to just panic
		panic(err)
	}
	// return a random database name prefixed with orders_test_ and suffixed with
	// the hexadecimal formatting of the random bytes
	return fmt.Sprintf("orders_test_%x", b)
}

////////////////////////////////////////////////////////////////////////////////

func TestGetOrder(t *testing.T) {
	// the context isn't meaningful for these tests so we just use a new one
	ctx := context.Background()
	// make a new instance with a random database so this test is isolated from
	// the others
	inst := New(randomDatabase())
	order := Order{
		ID:            "test",
		CustomerEmail: "test@test",
		LineItems: []LineItem{
			{
				Description: "item 1",
				Quantity:    1,
				PriceCents:  1000,
			},
			{
				Description: "item 2",
				Quantity:    10,
				PriceCents:  5000,
			},
		},
		Status: OrderStatusCharged,
	}
	id, err := inst.InsertOrder(ctx, order)
	// the require package fails the whole test immediately if this fails which is
	// useful for unexpected errors since the rest of the test will presumably fail
	// if we can't do this
	require.NoError(t, err)

	// returns expected order
	got, err := inst.GetOrder(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, order, got)

	// returns not found
	_, err = inst.GetOrder(ctx, "not found")
	// assert.Equal returns true if the assertion passes so we can use that as
	// a conditional around dependent tests so we don't end up having a bunch of
	// failed assertions
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, ErrOrderNotFound), "%#v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestGetOrders(t *testing.T) {
	// the context isn't meaningful for these tests so we just use a new one
	ctx := context.Background()
	// make a new instance with a random database so this test is isolated from
	// the others
	inst := New(randomDatabase())
	order1 := Order{
		ID:            "test1",
		CustomerEmail: "test@test",
		LineItems: []LineItem{
			{
				Description: "item 1",
				Quantity:    1,
				PriceCents:  1000,
			},
			{
				Description: "item 2",
				Quantity:    10,
				PriceCents:  5000,
			},
		},
		Status: OrderStatusCharged,
	}
	_, err := inst.InsertOrder(ctx, order1)
	// the require package fails the whole test immediately if this fails which is
	// useful for unexpected errors since the rest of the test will presumably fail
	// if we can't do this
	require.NoError(t, err)

	order2 := Order{
		ID:            "test2",
		CustomerEmail: "test@test",
		LineItems: []LineItem{
			{
				Description: "item 3",
				Quantity:    2,
				PriceCents:  500,
			},
			{
				Description: "item 4",
				Quantity:    1,
				PriceCents:  1000,
			},
		},
		Status: OrderStatusFulfilled,
	}
	_, err = inst.InsertOrder(ctx, order2)
	require.NoError(t, err)

	// returns all if -1 is sent
	got, err := inst.GetOrders(ctx, -1)
	require.NoError(t, err)
	// assert.Equal returns true if the assertion passes so we can use that as
	// a conditional around dependent tests so we don't end up having a bunch of
	// failed assertions
	if assert.Len(t, got, 2) {
		assert.Contains(t, got, order1)
		assert.Contains(t, got, order2)
	}

	// only returns the matching status
	got, err = inst.GetOrders(ctx, OrderStatusCharged)
	require.NoError(t, err)
	if assert.Len(t, got, 1) {
		assert.Contains(t, got, order1)
	}

	// only returns the matching status
	got, err = inst.GetOrders(ctx, OrderStatusFulfilled)
	require.NoError(t, err)
	if assert.Len(t, got, 1) {
		assert.Contains(t, got, order2)
	}

	// returns none and no error if none match
	got, err = inst.GetOrders(ctx, OrderStatusPending)
	require.NoError(t, err)
	assert.Empty(t, got)
}

////////////////////////////////////////////////////////////////////////////////

func TestSetOrderStatus(t *testing.T) {
	// the context isn't meaningful for these tests so we just use a new one
	ctx := context.Background()
	// make a new instance with a random database so this test is isolated from
	// the others
	inst := New(randomDatabase())
	id, err := inst.InsertOrder(ctx, Order{
		ID:            "test1",
		CustomerEmail: "test@test",
		LineItems: []LineItem{
			{
				Description: "item 1",
				Quantity:    1,
				PriceCents:  1000,
			},
			{
				Description: "item 2",
				Quantity:    10,
				PriceCents:  5000,
			},
		},
		Status: OrderStatusCharged,
	})
	// the require package fails the whole test immediately if this fails which is
	// useful for unexpected errors since the rest of the test will presumably fail
	// if we can't do this
	require.NoError(t, err)

	// returns all if -1 is sent
	err = inst.SetOrderStatus(ctx, id, OrderStatusFulfilled)
	require.NoError(t, err)

	got, err := inst.GetOrder(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, OrderStatusFulfilled, got.Status)

	// returns not found
	err = inst.SetOrderStatus(ctx, "not found", OrderStatusFulfilled)
	// assert.Equal returns true if the assertion passes so we can use that as
	// a conditional around dependent tests so we don't end up having a bunch of
	// failed assertions
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, ErrOrderNotFound), "%#v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestInsertOrder(t *testing.T) {
	// the context isn't meaningful for these tests so we just use a new one
	ctx := context.Background()
	// make a new instance with a random database so this test is isolated from
	// the others
	inst := New(randomDatabase())
	order1 := Order{
		ID:            "test1",
		CustomerEmail: "test@test",
		LineItems: []LineItem{
			{
				Description: "item 1",
				Quantity:    1,
				PriceCents:  1000,
			},
			{
				Description: "item 2",
				Quantity:    10,
				PriceCents:  5000,
			},
		},
		Status: OrderStatusCharged,
	}
	id, err := inst.InsertOrder(ctx, order1)
	// the require package fails the whole test immediately if this fails which is
	// useful for unexpected errors since the rest of the test will presumably fail
	// if we can't do this
	require.NoError(t, err)
	assert.Equal(t, order1.ID, id)

	// returns exists
	_, err = inst.InsertOrder(ctx, order1)
	// assert.Equal returns true if the assertion passes so we can use that as
	// a conditional around dependent tests so we don't end up having a bunch of
	// failed assertions
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, ErrOrderExists), "%#v", err)
	}

	// fills in an ID
	order2 := Order{
		CustomerEmail: "test@test",
		Status:        OrderStatusCharged,
	}
	id, err = inst.InsertOrder(ctx, order2)
	require.NoError(t, err)
	if assert.NotEmpty(t, id) {
		order2.ID = id

		got, err := inst.GetOrder(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, order2, got)
	}
}
