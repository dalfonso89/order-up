// Package api exposes an HTTP handler to handle REST API calls for manipulating
// and retrieving orders
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/levenlabs/order-up/mocks"
	"github.com/levenlabs/order-up/storage"
)

// instance represents an API instance. Typically this is exported but for our
// purposes we don't need to actually expose any methods on it since we only
// return an http.Handler implementation.
type instance struct {
	stor               mocks.StorageInstance
	router             *gin.Engine
	fulfillmentService *http.Client
	chargeService      *http.Client
}

// Handler returns an implementation of the http.Handler interface that can be
// passed to an http.Server to handle incoming HTTP requests. This accepts
// an interface for the storage.Instance and http.Client's for the 2 dependent
// services. Typically this would accept just a *storage.Instance but the mock
// allows us to separate the api tests from the storage tests.
func Handler(stor mocks.StorageInstance, fulfillmentService, chargeService *http.Client) http.Handler {
	// inst is pointer to a new instance that's holding a new storage.Instance for
	// talking to the underlying database
	inst := &instance{
		stor:               stor,
		router:             gin.Default(),
		fulfillmentService: fulfillmentService,
		chargeService:      chargeService,
	}

	// Add logging middleware to all routes
	inst.router.Use(inst.loggingMiddleware())

	// set up the various REST endpoints that are exposed publicly over HTTP
	// go implicitly binds these functions to inst
	inst.router.GET("/healthz", inst.healthCheck)
	inst.router.GET("/orders", inst.getOrders)
	inst.router.POST("/orders", inst.postOrders)

	// Use order fetch middleware for routes that need to fetch an order
	inst.router.GET("/orders/:id", inst.orderFetchMiddleware(), inst.getOrder)
	inst.router.POST("/orders/:id/charge", inst.orderFetchMiddleware(), inst.chargeOrder)
	inst.router.POST("/orders/:id/cancel", inst.orderFetchMiddleware(), inst.cancelOrder)

	// *instance implements the http.Handler interface with the ServeHTTP method
	// below so we can just return inst
	return inst
}

// ServeHTTP implements the http.Handler interface and passes incoming HTTP
// requests to the underlying *gin.Engine
func (i *instance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i.router.ServeHTTP(w, r)
}

////////////////////////////////////////////////////////////////////////////////

type getOrdersRes struct {
	Orders []storage.Order `json:"orders"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error codes for different types of errors
const (
	ErrCodeOrderNotFound      = "order_not_found"
	ErrCodeOrderExists        = "order_already_exists"
	ErrCodeInvalidEmail       = "invalid_email"
	ErrCodeInvalidLineItems   = "invalid_line_items"
	ErrCodeInvalidTotal       = "invalid_total"
	ErrCodeInvalidStatus      = "invalid_status"
	ErrCodeOrderNotCharged    = "order_not_charged"
	ErrCodeOrderNotEligible   = "order_not_eligible"
	ErrCodeInvalidJSON        = "invalid_json"
	ErrCodeInternalError      = "internal_error"
	ErrCodeChargeServiceError = "charge_service_error"
)

// Helper functions for creating structured errors
func (i *instance) handleError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, errorResponse{
		Code:    code,
		Message: message,
	})
}

// Middleware for centralized logging
func (i *instance) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	})
}

// Middleware for handling order fetching with centralized error handling
func (i *instance) orderFetchMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Get order ID from URL parameter
		id := c.Param("id")
		if id == "" {
			i.handleError(c, http.StatusBadRequest, ErrCodeInvalidStatus, "missing order ID")
			c.Abort()
			return
		}

		// Fetch order from storage
		order, err := i.stor.GetOrder(ctx, id)
		if err != nil {
			if errors.Is(err, storage.ErrOrderNotFound) {
				i.handleError(c, http.StatusNotFound, ErrCodeOrderNotFound, "not found")
			} else {
				i.handleError(c, http.StatusInternalServerError, ErrCodeInternalError, fmt.Sprintf("error getting order: %v", err))
			}
			c.Abort()
			return
		}

		// Store order in context for use by handlers
		c.Set("order", order)
		c.Next()
	}
}

// Helper function to get order from context (set by middleware)
func (i *instance) getOrderFromContext(c *gin.Context) storage.Order {
	order, _ := c.Get("order")
	return order.(storage.Order)
}

// healthCheck is called by incoming HTTP GET requests to /healthz
func (i *instance) healthCheck(c *gin.Context) {
	c.Status(http.StatusOK)
}

// getOrders is called by incoming HTTP GET requests to /orders
func (i *instance) getOrders(c *gin.Context) {
	// the context of the request we pass along to every downstream function so we
	// can stop processing if the caller aborts the request and also to ensure that
	// the tracing context is kept throughout the whole request
	ctx := c.Request.Context()

	// get and parse the optional status query parameter from the request
	// this lets you do /orders?status=pending to limit the orders to only those that
	// are currently pending
	var status storage.OrderStatus
	switch c.Query("status") {
	case "pending":
		status = storage.OrderStatusPending
		// the final break is implied if there's no fallthrough keyword
	case "charged":
		status = storage.OrderStatusCharged
	case "fulfilled":
		status = storage.OrderStatusFulfilled
	case "cancelled":
		status = storage.OrderStatusCancelled
	case "":
		// GetAllOrders accepts a -1 to indicate that all orders should be returned
		status = -1
	default:
		i.handleError(c, http.StatusBadRequest, ErrCodeInvalidStatus, "unknown value for status: %v")
		return
	}

	// pass along the status and get all of the resulting orders from the storage
	// instance
	orders, err := i.stor.GetOrders(ctx, status)
	if err != nil {
		i.handleError(c, http.StatusInternalServerError, ErrCodeInternalError, fmt.Sprintf("error getting orders: %v", err))
		return
	}

	// by default slices are nil and if we return that the resulting JSON would be
	// {"orders":null} which some languages/clients have a problem with
	// instead set it to an empty slice
	if orders == nil {
		orders = []storage.Order{}
	}

	// respond with a success and return the orders
	c.JSON(http.StatusOK, getOrdersRes{
		Orders: orders,
	})
}

////////////////////////////////////////////////////////////////////////////////

// getOrderRes is the result of the GET /orders/:id handler
// you might think its unnecessary for this struct and we should instead just
// return the order itself but this gives us future flexibility to return
// anything else alongside that we can't think of right now
type getOrderRes struct {
	Order storage.Order `json:"order"`
}

// getOrder is called by incoming HTTP GET requests to /orders/:id
func (i *instance) getOrder(c *gin.Context) {
	// Get order from context (set by middleware)
	order := i.getOrderFromContext(c)

	// respond with a success and return the order
	c.JSON(http.StatusOK, getOrderRes{
		Order: order,
	})
}

////////////////////////////////////////////////////////////////////////////////

// postOrderArgs is the expected body for the POST /orders handler
type postOrderArgs struct {
	CustomerEmail string             `json:"customerEmail"`
	LineItems     []storage.LineItem `json:"lineItems"`
}

// chargeOrderRes is the result of the POST /orders/:id/charge handler
type postOrderRes struct {
	Order storage.Order `json:"order"`
}

// postOrders is called by incoming HTTP POST requests to /orders
func (i *instance) postOrders(c *gin.Context) {
	// the context of the request we pass along to every downstream function so we
	// can stop processing if the caller aborts the request and also to ensure that
	// the tracing context is kept throughout the whole request
	ctx := c.Request.Context()

	// parse the body as JSON into the newOrderArgs struct
	var args postOrderArgs
	err := c.BindJSON(&args)
	if err != nil {
		i.handleError(c, http.StatusBadRequest, ErrCodeInvalidJSON, fmt.Sprintf("error decoding body: %v", err))
		return
	}

	// do some light validation
	// we could use something like https://pkg.go.dev/gopkg.in/validator.v2
	// so we could set struct tags but since we only do validation in this one
	// spot that feels like overkill
	if !strings.Contains(args.CustomerEmail, "@") {
		i.handleError(c, http.StatusBadRequest, ErrCodeInvalidEmail, "invalid customerEmail")
		return
	}
	if len(args.LineItems) < 1 {
		i.handleError(c, http.StatusBadRequest, ErrCodeInvalidLineItems, "an order must contain at least one line item")
		return
	}

	order := storage.Order{
		CustomerEmail: args.CustomerEmail,
		LineItems:     args.LineItems,
		Status:        storage.OrderStatusPending,
	}
	if order.TotalCents() < 0 {
		i.handleError(c, http.StatusBadRequest, ErrCodeInvalidTotal, "an order's total cannot be less than 0")
		return
	}

	id, err := i.stor.InsertOrder(ctx, order)
	if err != nil {
		if errors.Is(err, storage.ErrOrderExists) {
			i.handleError(c, http.StatusConflict, ErrCodeOrderExists, "order already exists")
		} else {
			i.handleError(c, http.StatusInternalServerError, ErrCodeInternalError, fmt.Sprintf("error inserting order: %v", err))
		}
		return
	}
	order.ID = id

	// respond with a success and return the order
	c.JSON(http.StatusCreated, postOrderRes{
		Order: order,
	})
}

////////////////////////////////////////////////////////////////////////////////

// chargeOrderArgs is the expected body for the POST /orders/:id/charge handler
type chargeOrderArgs struct {
	CardToken string `json:"cardToken"`
}

// chargeServiceChargeArgs is the expected body for the charge service
type chargeServiceChargeArgs struct {
	CardToken   string `json:"cardToken"`
	AmountCents int64  `json:"amountCents"`
}

// fulfillmentServiceFulfillArgs is the expected body for the fulfillment service
type fulfillmentServiceFulfillArgs struct {
	Description string `json:"description"`
	Quantity    int64  `json:"quantity"`
	OrderID     string `json:"orderId"`
}

// chargeOrderRes is the result of the POST /orders/:id/charge handler
type chargeOrderRes struct {
	ChargedCents int64 `json:"chargedCents"`
}

// chargeOrder is called by incoming HTTP POST requests to /orders/:id/charge
func (i *instance) chargeOrder(c *gin.Context) {
	ctx := c.Request.Context()

	// parse the body as JSON into the chargeOrderArgs struct
	var args chargeOrderArgs
	err := c.BindJSON(&args)
	if err != nil {
		i.handleError(c, http.StatusBadRequest, ErrCodeInvalidJSON, fmt.Sprintf("error decoding body: %v", err))
		return
	}

	// Get order from context (set by middleware)
	order := i.getOrderFromContext(c)

	if order.Status != storage.OrderStatusPending {
		i.handleError(c, http.StatusConflict, ErrCodeOrderNotEligible,
			"order ineligible for charging")
		return
	}

	err = i.innerChargeOrder(ctx, chargeServiceChargeArgs{
		CardToken:   args.CardToken,
		AmountCents: order.TotalCents(),
	})
	if err != nil {
		i.handleError(c, http.StatusInternalServerError, ErrCodeChargeServiceError,
			err.Error())
		return
	}

	// in a real-world scenario we would do a two-phase change where we set it to
	// charging ahead of time and then mark it as charged after so we would be able
	// to understand if this was retried that we already tried to charge
	// as it's written if this service crashed before this line then we would've
	// charged the customer and not reflected that on the order but for now we're
	// ignoring this scenario
	err = i.stor.SetOrderStatus(ctx, order.ID, storage.OrderStatusCharged)
	if err != nil {
		i.handleError(c, http.StatusInternalServerError, ErrCodeInternalError, fmt.Sprintf("error updating order to charged: %v", err))
		return
	}

	// since we successfully charged the order and updated the order status we can
	// return a success to the caller
	c.JSON(http.StatusOK, chargeOrderRes{
		ChargedCents: order.TotalCents(),
	})
}

// innerChargeOrder actually does the charging or refunding (negative amount) by
// making at POST request to the charge service
func (i *instance) innerChargeOrder(ctx context.Context, args chargeServiceChargeArgs) error {
	// encode the charge service's charge arguments as JSON so we can POST them to
	// the /charge path on the charge service
	// this method returns a byte slice that we can later pass to the Post message
	// as the body of the POST request
	// there's a package called "bytes" so we call the variable byts
	byts, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("error encoding charge body: %w", err)
	}

	// make a POST request to the /charge endpoint on the charge service
	// the body is JSON but this method accepts a io.Reader so we need to wrap the
	// byte slice in bytes.NewReader which simply reads over the sent byte slice
	resp, err := i.chargeService.Post("/charge", "application/json", bytes.NewReader(byts))
	if err != nil {
		return fmt.Errorf("error making charge request: %w", err)
	}
	// we need to make sure we close the body otherwise this will leak memory
	defer resp.Body.Close()
	// /charge creates a new charge so we expect a 201 response, if we didn't get
	// that then we must've errored
	if resp.StatusCode != http.StatusCreated {
		// we opportunistically try to read the body in case it contains an error but
		// if it fails then that's not the end of the world so we ignore the error
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("error charging body: %d %s", resp.StatusCode, body)
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// cancelOrderRes is the result of the POST /orders/:id/cancel handler
type cancelOrderRes struct {
	Message       string `json:"message"`
	OrderID       string `json:"orderId"`
	RefundedCents int64  `json:"refundedCents,omitempty"`
}

// cancelOrder is called by incoming HTTP POST requests to /orders/:id/cancel
func (i *instance) cancelOrder(c *gin.Context) {
	ctx := c.Request.Context()

	// Get order from context (set by middleware)
	order := i.getOrderFromContext(c)

	// Check if order can be cancelled (pending or charged orders can be cancelled)
	if order.Status != storage.OrderStatusPending && order.Status != storage.OrderStatusCharged {
		i.handleError(c, http.StatusConflict, ErrCodeOrderNotEligible,
			"order cannot be cancelled - only pending or charged orders can be cancelled")
		return
	}

	var refundedCents int64 = 0

	// If the order is charged, we need to process a refund
	if order.Status == storage.OrderStatusCharged {
		// Process refund by charging a negative amount
		err := i.innerChargeOrder(ctx, chargeServiceChargeArgs{
			CardToken:   "",                  // In a real implementation, we'd need to store the card token
			AmountCents: -order.TotalCents(), // Negative amount for refund
		})
		if err != nil {
			i.handleError(c, http.StatusInternalServerError, ErrCodeChargeServiceError,
				fmt.Sprintf("error processing refund: %v", err))
			return
		}
		refundedCents = order.TotalCents()
	}

	// Update order status to cancelled
	err := i.stor.SetOrderStatus(ctx, order.ID, storage.OrderStatusCancelled)
	if err != nil {
		i.handleError(c, http.StatusInternalServerError, ErrCodeInternalError,
			fmt.Sprintf("error cancelling order: %v", err))
		return
	}

	// Return success response
	response := cancelOrderRes{
		Message: "order cancelled successfully",
		OrderID: order.ID,
	}

	// Include refund amount if applicable
	if refundedCents > 0 {
		response.RefundedCents = refundedCents
	}

	c.JSON(http.StatusOK, response)
}
