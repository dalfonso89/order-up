package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/levenlabs/order-up/mocks"
	"github.com/levenlabs/order-up/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// init is a special function that's automatically run when this package is imported
func init() {
	// gin's debug mode is very noisy so we set it to release mode
	// if you're debugging the handler you might want to comment this out
	gin.SetMode(gin.TestMode)
}

////////////////////////////////////////////////////////////////////////////////

func TestGetOrders(t *testing.T) {
	// the context just needs to be something static so we can include it in the
	// mocked arguments
	ctx := context.Background()

	// these braces form a new scope so we don't end up polluting the top-level
	// function with our recorder, request, etc
	// they also visually break up the inner tests

	// shouldn't return any errors if none were found
	{
		// make a pointer to a mocks.MockStorageInstance struct which is necessary
		// for mocking the storage package
		stor := new(mocks.MockStorageInstance)
		// On queues up a new expected call with the provided arguments and returns
		// the values sent to Return
		// we also only expect this call to only happen Once
		stor.On("GetOrders", ctx, storage.OrderStatus(-1)).Return([]storage.Order{}, nil).Once()
		// we know that this call doesn't make any external calls so we can just pass
		// nil to simplify this code
		h := Handler(stor, nil, nil)
		// httptest is a package to help with testing http servers
		// NewRecorder returns an http.ResponseWriter that allows us to record the
		// status and body set by the caller
		w := httptest.NewRecorder()
		// httptest.NewRequest is like http.NewRequest except that it doesn't return
		// an error and panic's if an error would've been returned
		// we can pass nil as the body here since it's a GET request without any body
		r := httptest.NewRequest("GET", "/orders", nil).WithContext(ctx)
		h.ServeHTTP(w, r)
		// assert.Equal returns true if the assertion passes so we can use that as
		// a conditional around dependent tests so we don't end up having a bunch of
		// failed assertions
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.HeaderMap.Get("Content-Type"), "application/json")
			// in this case we can just test the exact string rather than bothering to
			// JSON unmarshal it
			assert.Equal(t, `{"orders":[]}`, w.Body.String())
		}
		stor.AssertExpectations(t)
	}

	// define some orders just to make it easier later
	order1 := storage.Order{
		ID:        "test1",
		LineItems: []storage.LineItem{},
		Status:    storage.OrderStatusCharged,
	}
	order2 := storage.Order{
		ID:        "test2",
		LineItems: []storage.LineItem{},
		Status:    storage.OrderStatusFulfilled,
	}

	// should return all orders
	{
		stor := new(mocks.MockStorageInstance)
		stor.On("GetOrders", ctx, storage.OrderStatus(-1)).Return([]storage.Order{order1, order2}, nil).Once()
		h := Handler(stor, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/orders", nil).WithContext(ctx)
		h.ServeHTTP(w, r)
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.HeaderMap.Get("Content-Type"), "application/json")
			var res getOrdersRes
			err := json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)
			if assert.Len(t, res.Orders, 2) {
				assert.Contains(t, res.Orders, order1)
				assert.Contains(t, res.Orders, order2)
			}
		}
		stor.AssertExpectations(t)
	}

	// should return charged orders
	{
		stor := new(mocks.MockStorageInstance)
		stor.On("GetOrders", ctx, storage.OrderStatusCharged).Return([]storage.Order{order1}, nil).Once()
		h := Handler(stor, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/orders?status=charged", nil).WithContext(ctx)
		h.ServeHTTP(w, r)
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.HeaderMap.Get("Content-Type"), "application/json")
			var res getOrdersRes
			err := json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)
			if assert.Len(t, res.Orders, 1) {
				assert.Contains(t, res.Orders, order1)
			}
		}
		stor.AssertExpectations(t)
	}

	// should return pending orders
	{
		stor := new(mocks.MockStorageInstance)
		stor.On("GetOrders", ctx, storage.OrderStatusPending).Return([]storage.Order{}, nil).Once()
		h := Handler(stor, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/orders?status=pending", nil).WithContext(ctx)
		h.ServeHTTP(w, r)
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.HeaderMap.Get("Content-Type"), "application/json")
			var res getOrdersRes
			err := json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)
			assert.Empty(t, res.Orders)
		}
		stor.AssertExpectations(t)
	}

	// should error on unknown status
	{
		stor := new(mocks.MockStorageInstance)
		h := Handler(stor, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/orders?status=unknown", nil).WithContext(ctx)
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		stor.AssertExpectations(t)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestGetOrder(t *testing.T) {
	// the context just needs to be something static so we can include it in the
	// mocked arguments
	ctx := context.Background()

	// these braces form a new scope so we don't end up polluting the top-level
	// function with our recorder, request, etc
	// they also visually break up the inner tests

	// should return 404 on not found
	{
		// make a pointer to a mocks.MockStorageInstance struct which is necessary
		// for mocking the storage package
		stor := new(mocks.MockStorageInstance)
		// On queues up a new expected call with the provided arguments and returns
		// the values sent to Return
		// we also only expect this call to only happen Once
		stor.On("GetOrder", ctx, "notfound").Return(storage.Order{}, storage.ErrOrderNotFound).Once()
		// we know that this call doesn't make any external calls so we can just pass
		// nil to simplify this code
		h := Handler(stor, nil, nil)
		// httptest is a package to help with testing http servers
		// NewRecorder returns an http.ResponseWriter that allows us to record the
		// status and body set by the caller
		w := httptest.NewRecorder()
		// httptest.NewRequest is like http.NewRequest except that it doesn't return
		// an error and panic's if an error would've been returned
		// we can pass nil as the body here since it's a GET request without any body
		r := httptest.NewRequest("GET", "/orders/notfound", nil).WithContext(ctx)
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
		stor.AssertExpectations(t)
	}

	// insert an order so we can have something to retrieve
	// we can just use the minimal fields since we're not testing the whole storage
	// package just the filtering
	order1 := storage.Order{
		ID:        "test1",
		LineItems: []storage.LineItem{},
		Status:    storage.OrderStatusCharged,
	}

	// should return the above order
	{
		stor := new(mocks.MockStorageInstance)
		stor.On("GetOrder", ctx, order1.ID).Return(order1, nil).Once()
		h := Handler(stor, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path.Join("/orders", order1.ID), nil).WithContext(ctx)
		h.ServeHTTP(w, r)
		// assert.Equal returns true if the assertion passes so we can use that as
		// a conditional around dependent tests so we don't end up having a bunch of
		// failed assertions
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.HeaderMap.Get("Content-Type"), "application/json")
			var res getOrderRes
			err := json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)
			assert.Equal(t, order1, res.Order)
		}
		stor.AssertExpectations(t)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestPostOrders(t *testing.T) {
	// the context just needs to be something static so we can include it in the
	// mocked arguments
	ctx := context.Background()

	// these braces form a new scope so we don't end up polluting the top-level
	// function with our recorder, request, etc
	// they also visually break up the inner tests

	// successfully inserts a valid order
	{
		id := "random"
		expOrder := storage.Order{
			CustomerEmail: "test@test",
			LineItems: []storage.LineItem{
				{
					Description: "item 1",
					Quantity:    1,
					PriceCents:  1000,
				},
			},
			Status: storage.OrderStatusPending,
		}
		args := postOrderArgs{
			CustomerEmail: expOrder.CustomerEmail,
			LineItems:     expOrder.LineItems,
		}
		// make a pointer to a mocks.MockStorageInstance struct which is necessary
		// for mocking the storage package
		stor := new(mocks.MockStorageInstance)
		// On queues up a new expected call with the provided arguments and returns
		// the values sent to Return
		// we also only expect this call to only happen Once
		stor.On("InsertOrder", ctx, expOrder).Return(id, nil).Once()
		// we know that this call doesn't make any external calls so we can just pass
		// nil to simplify this code
		h := Handler(stor, nil, nil)
		// httptest is a package to help with testing http servers
		// NewRecorder returns an http.ResponseWriter that allows us to record the
		// status and body set by the caller
		w := httptest.NewRecorder()
		// httptest.NewRequest is like http.NewRequest except that it doesn't return
		// an error and panic's if an error would've been returned
		// encode the POST /orders arguments as JSON so we can include them in the
		// body for the request
		// there's a package called "bytes" so we call the variable byts
		byts, err := json.Marshal(args)
		// the require package fails the whole test immediately if this fails which is
		// useful for unexpected errors since the rest of the test will presumably fail
		// if we can't do this
		require.NoError(t, err)
		// the body is JSON but this method accepts a io.Reader so we need to wrap
		// the byte slice in bytes.NewReader which simply reads over the sent byte
		// slice
		r := httptest.NewRequest("POST", "/orders", bytes.NewReader(byts)).WithContext(ctx)
		h.ServeHTTP(w, r)
		// assert.Equal returns true if the assertion passes so we can use that as
		// a conditional around dependent tests so we don't end up having a bunch of
		// failed assertions
		if assert.Equal(t, http.StatusCreated, w.Code) {
			assert.Contains(t, w.HeaderMap.Get("Content-Type"), "application/json")
			var res postOrderRes
			err = json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)
			// set the ID since that happens inside of the handler
			expOrder.ID = id
			assert.Equal(t, expOrder, res.Order)
		}
		stor.AssertExpectations(t)
	}

	// should error on invalid email
	{
		stor := new(mocks.MockStorageInstance)
		h := Handler(stor, nil, nil)
		w := httptest.NewRecorder()
		byts, err := json.Marshal(postOrderArgs{
			CustomerEmail: "invalid",
			LineItems: []storage.LineItem{
				{
					Description: "item 1",
					Quantity:    1,
					PriceCents:  1000,
				},
			},
		})
		require.NoError(t, err)
		r := httptest.NewRequest("POST", "/orders", bytes.NewReader(byts)).WithContext(ctx)
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		stor.AssertExpectations(t)
	}

	// should error on no line items
	{
		stor := new(mocks.MockStorageInstance)
		h := Handler(stor, nil, nil)
		w := httptest.NewRecorder()
		byts, err := json.Marshal(postOrderArgs{
			CustomerEmail: "test@test",
		})
		require.NoError(t, err)
		r := httptest.NewRequest("POST", "/orders", bytes.NewReader(byts)).WithContext(ctx)
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		stor.AssertExpectations(t)
	}

	// should error on negative order amount
	{
		stor := new(mocks.MockStorageInstance)
		h := Handler(stor, nil, nil)
		w := httptest.NewRecorder()
		byts, err := json.Marshal(postOrderArgs{
			CustomerEmail: "test@test",
			LineItems: []storage.LineItem{
				{
					Description: "item 1",
					Quantity:    1,
					PriceCents:  1000,
				},
				{
					Description: "huge discount",
					Quantity:    1,
					PriceCents:  -10000,
				},
			},
		})
		require.NoError(t, err)
		r := httptest.NewRequest("POST", "/orders", bytes.NewReader(byts)).WithContext(ctx)
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		stor.AssertExpectations(t)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestChargeOrder(t *testing.T) {
	// the context just needs to be something static so we can include it in the
	// mocked arguments
	ctx := context.Background()

	var chgServCalled int64
	chgServ := mocks.NewMockedService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure the URL is /charge and the method is POST since that's the only
		// endpoint the charge service has
		require.Equal(t, "/charge", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		// decode the body as a chargeServiceChargeArgs
		var args chargeServiceChargeArgs
		err := json.NewDecoder(r.Body).Decode(&args)
		require.NoError(t, err)

		// make sure the args are sane
		require.True(t, args.AmountCents > 0, "amountCents must be more than 0: %v", args.AmountCents)
		require.NotEmpty(t, args.CardToken)

		// increment calls so we can test to make sure the charge service was ever
		// called and that it was only called an expected number of times
		atomic.AddInt64(&chgServCalled, 1)
		w.WriteHeader(http.StatusCreated)
	}))

	// these braces form a new scope so we don't end up polluting the top-level
	// function with our recorder, request, etc
	// they also visually break up the inner tests

	// should charge update order status
	{
		chgServCalled = 0
		order := storage.Order{
			ID:            "test",
			CustomerEmail: "test@test",
			LineItems: []storage.LineItem{
				{
					Description: "item 1",
					Quantity:    1,
					PriceCents:  100,
				},
			},
			Status: storage.OrderStatusPending,
		}
		args := chargeOrderArgs{
			CardToken: "amex",
		}
		// make a pointer to a mocks.MockStorageInstance struct which is necessary
		// for mocking the storage package
		stor := new(mocks.MockStorageInstance)
		// On queues up a new expected call with the provided arguments and returns
		// the values sent to Return
		// we also only expect this call to only happen Once
		stor.On("GetOrder", ctx, order.ID).Return(order, nil).Once()
		stor.On("SetOrderStatus", ctx, order.ID, storage.OrderStatusCharged).Return(nil).Once()
		// no need to pass along a fulfillment service since we know we're only
		// calling storage and charge service
		h := Handler(stor, nil, chgServ)
		// httptest is a package to help with testing http servers
		// NewRecorder returns an http.ResponseWriter that allows us to record the
		// status and body set by the caller
		w := httptest.NewRecorder()
		// httptest.NewRequest is like http.NewRequest except that it doesn't return
		// an error and panic's if an error would've been returned
		// encode the POST /orders arguments as JSON so we can include them in the
		// body for the request
		// there's a package called "bytes" so we call the variable byts
		byts, err := json.Marshal(args)
		// the require package fails the whole test immediately if this fails which is
		// useful for unexpected errors since the rest of the test will presumably fail
		// if we can't do this
		require.NoError(t, err)
		// the body is JSON but this method accepts a io.Reader so we need to wrap
		// the byte slice in bytes.NewReader which simply reads over the sent byte
		// slice
		r := httptest.NewRequest("POST", path.Join("/orders", order.ID, "charge"), bytes.NewReader(byts)).WithContext(ctx)
		h.ServeHTTP(w, r)
		// assert.Equal returns true if the assertion passes so we can use that as
		// a conditional around dependent tests so we don't end up having a bunch of
		// failed assertions
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.HeaderMap.Get("Content-Type"), "application/json")
			var res chargeOrderRes
			err = json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)
			// assert.Equal tests the value AND type but if we just write "100" that's
			// an int but ChargedCents is an int64, by instead using assert.EqualValues
			// we ignore the type and only compare the values
			// alternatively we could write assert.Equal(t, int64(100), res.ChargedCents)
			assert.EqualValues(t, 100, res.ChargedCents)
			assert.EqualValues(t, 1, chgServCalled)
		}
		stor.AssertExpectations(t)
	}

	// should error and skip charging if already charged
	{
		chgServCalled = 0
		order := storage.Order{
			ID:            "test",
			CustomerEmail: "test@test",
			LineItems: []storage.LineItem{
				{
					Description: "item 1",
					Quantity:    1,
					PriceCents:  100,
				},
			},
			Status: storage.OrderStatusCharged,
		}
		args := chargeOrderArgs{
			CardToken: "amex",
		}
		stor := new(mocks.MockStorageInstance)
		stor.On("GetOrder", ctx, order.ID).Return(order, nil).Once()
		h := Handler(stor, nil, chgServ)
		w := httptest.NewRecorder()
		byts, err := json.Marshal(args)
		require.NoError(t, err)
		r := httptest.NewRequest("POST", path.Join("/orders", order.ID, "charge"), bytes.NewReader(byts)).WithContext(ctx)
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.EqualValues(t, 0, chgServCalled)
		stor.AssertExpectations(t)
	}

	// should error and skip charging if already fulfilled
	{
		chgServCalled = 0
		order := storage.Order{
			ID:            "test",
			CustomerEmail: "test@test",
			LineItems: []storage.LineItem{
				{
					Description: "item 1",
					Quantity:    1,
					PriceCents:  100,
				},
			},
			Status: storage.OrderStatusFulfilled,
		}
		args := chargeOrderArgs{
			CardToken: "amex",
		}
		stor := new(mocks.MockStorageInstance)
		stor.On("GetOrder", ctx, order.ID).Return(order, nil).Once()
		h := Handler(stor, nil, chgServ)
		w := httptest.NewRecorder()
		byts, err := json.Marshal(args)
		require.NoError(t, err)
		r := httptest.NewRequest("POST", path.Join("/orders", order.ID, "charge"), bytes.NewReader(byts)).WithContext(ctx)
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.EqualValues(t, 0, chgServCalled)
		stor.AssertExpectations(t)
	}

	// should skip charging if no amount is due but update order status
	{
		chgServCalled = 0
		order := storage.Order{
			ID:            "test",
			CustomerEmail: "test@test",
			LineItems: []storage.LineItem{
				{
					Description: "item 1",
					Quantity:    1,
					PriceCents:  100,
				},
				{
					Description: "#1 customer discount",
					Quantity:    1,
					PriceCents:  -100,
				},
			},
			Status: storage.OrderStatusPending,
		}
		args := chargeOrderArgs{
			CardToken: "amex",
		}
		stor := new(mocks.MockStorageInstance)
		stor.On("GetOrder", ctx, order.ID).Return(order, nil).Once()
		stor.On("SetOrderStatus", ctx, order.ID, storage.OrderStatusCharged).Return(nil).Once()
		h := Handler(stor, nil, chgServ)
		w := httptest.NewRecorder()
		byts, err := json.Marshal(args)
		require.NoError(t, err)
		r := httptest.NewRequest("POST", path.Join("/orders", order.ID, "charge"), bytes.NewReader(byts)).WithContext(ctx)
		h.ServeHTTP(w, r)
		if assert.Equal(t, http.StatusOK, w.Code) {
			assert.Contains(t, w.HeaderMap.Get("Content-Type"), "application/json")
			var res chargeOrderRes
			err = json.Unmarshal(w.Body.Bytes(), &res)
			require.NoError(t, err)
			assert.EqualValues(t, 0, res.ChargedCents)
			assert.EqualValues(t, 0, chgServCalled)
		}
		stor.AssertExpectations(t)
	}

	// should not have more than 1 outstanding charge service request
	{
		chgServCalled = 0
		order := storage.Order{
			ID:            "test",
			CustomerEmail: "test@test",
			LineItems: []storage.LineItem{
				{
					Description: "item 1",
					Quantity:    1,
					PriceCents:  100,
				},
			},
			Status: storage.OrderStatusPending,
		}
		args := chargeOrderArgs{
			CardToken: "amex",
		}

		// we make a new chgServ mock that tracks the concurrency using the atomic
		// package
		var concurrent int64
		chgServ := mocks.NewMockedService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/charge", r.URL.Path)
			require.Equal(t, http.MethodPost, r.Method)

			// ensure that only 1 call is happening concurrently by adding 1 to concurrent
			// and checking to ensure it was 0 and is now 1
			// the atomic package allows us to avoid locking and its a little simpler to
			// work with in this situation
			running := atomic.AddInt64(&concurrent, 1)
			// defer decrementing the concurrent when this function is done running
			defer atomic.AddInt64(&concurrent, -1)
			require.EqualValues(t, 1, running, "detected more than 1 /charge happening concurrently")

			// goroutines do not start immediately and we can't control how go schedules
			// them so this sleep gives us a chance to try and see if another goroutine
			// ends up calling this while we're sleeping to try and detect concurrency
			// this isn't perfect but should be sufficient enough for this project
			time.Sleep(time.Second)

			// increment calls so we can test to make sure the charge service was ever
			// called and that it was only called an expected number of times
			atomic.AddInt64(&chgServCalled, 1)
			w.WriteHeader(http.StatusCreated)
		}))

		times := 5
		stor := new(mocks.MockStorageInstance)
		stor.On("GetOrder", ctx, order.ID).Return(order, nil).Times(times)
		stor.On("SetOrderStatus", ctx, order.ID, storage.OrderStatusCharged).Return(nil).Times(times)
		h := Handler(stor, nil, chgServ)

		// sync.WaitGroup is a handy tool for waiting until a bunch of goroutines
		// return
		// whenever you spawn a new goroutine you increment and whenever a goroutine
		// finishes you call done
		var wg sync.WaitGroup
		for i := 0; i < times; i++ {
			wg.Add(1)
			// each of these goroutines will make the same charge call
			// it's not important that they're all charging the same order for the
			// purposes of testing concurrency
			go func() {
				defer wg.Done()
				w := httptest.NewRecorder()
				byts, err := json.Marshal(args)
				require.NoError(t, err)
				r := httptest.NewRequest("POST", path.Join("/orders", order.ID, "charge"), bytes.NewReader(byts)).WithContext(ctx)
				h.ServeHTTP(w, r)
				assert.Equal(t, http.StatusOK, w.Code)
			}()
		}

		// wait until all of the goroutines are done
		wg.Wait()
		assert.EqualValues(t, times, chgServCalled)
		stor.AssertExpectations(t)
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestPostCancelOrder(t *testing.T) {
	// TODO: add tests
}

////////////////////////////////////////////////////////////////////////////////

func TestPostFulfillOrder(t *testing.T) {
	// the context just needs to be something static so we can include it in the
	// mocked arguments
	ctx := context.Background()

	// keep track of which items have already been fulfilled so we can ignore ones
	// that have already been requested
	fulfilledItems := map[string]bool{}
	var fulfilledItemsLock sync.Mutex
	// fulfillments will only be incremented if the fulfillment happened for the
	// first time so this can be used to test for deduplication of the actual
	// fulfillment service
	var fulfillments int64
	fulfillServ := mocks.NewMockedService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure the URL is /fulfill and the method is PUT since that's the only
		// endpoint the fufillment service has
		require.Equal(t, "/fulfill", r.URL.Path)
		require.Equal(t, http.MethodPut, r.Method)

		// decode the body as a fulfillmentServiceFulfillArgs
		var args fulfillmentServiceFulfillArgs
		err := json.NewDecoder(r.Body).Decode(&args)
		require.NoError(t, err)

		// make sure the args are sane
		require.NotEmpty(t, args.Description)
		require.True(t, args.Quantity > 0, "quantity must be more than 0: %v", args.Quantity)
		require.NotEmpty(t, args.OrderID)

		// we need to lock around a map since the map isn't safe for concurrent
		// reads/writes
		fulfilledItemsLock.Lock()
		// we want to make sure we ALWAYS unlock no matter
		defer fulfilledItemsLock.Unlock()

		// the "key" in the fulfilledItems map is the orderID and the description
		// this could probably be more complicated but that seems unnecessary
		key := args.OrderID + args.Description
		// if the map already contains a true value for that key then we can
		// immediately return since we've already fulfilled
		if fulfilledItems[key] {
			w.WriteHeader(http.StatusOK)
			return
		}
		// we are fulfilling for the first time so set to true and continue to
		// incrementing
		fulfilledItems[key] = true
		atomic.AddInt64(&fulfillments, 1)
		w.WriteHeader(http.StatusOK)
	}))

	// these 2 lines are strictly to prevent Go from complaining that we didn't use
	// these variables and you should delete these lines once you add the tests
	_ = ctx
	_ = fulfillServ
	// TODO: add tests

}
