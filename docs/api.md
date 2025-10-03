# Order Up API Documentation

This document describes the REST API for the Order Up service, which manages orders, payments, and order lifecycle.

## Base Information

- **Base URL**: `http://localhost:8888`
- **Content-Type**: `application/json`
- **API Version**: 1.0

## Overview

The Order Up API provides endpoints for:
- Order management (create, retrieve, update status)
- Payment processing (charge orders)
- Order lifecycle management (cancel orders, process refunds)
- Health monitoring

## Data Models

### OrderStatus Enum

Order status values:
- `0` - `pending`: Order created but not yet charged
- `1` - `charged`: Order has been successfully charged
- `2` - `fulfilled`: Order has been fulfilled and shipped
- `3` - `cancelled`: Order has been cancelled

### LineItem

Represents a single item or charge on an order.

```json
{
  "description": "string",
  "priceCents": "integer(int64)",
  "quantity": "integer(int64)"
}
```

- `description`: Product ID, discount ID, or item description
- `priceCents`: Individual price in cents (can be negative for discounts)
- `quantity`: Number of items (always positive)

### Order

Complete order information.

```json
{
  "id": "string",
  "customerEmail": "string",
  "lineItems": [
    {
      "description": "string",
      "priceCents": "integer(int64)",
      "quantity": "integer(int64)"
    }
  ],
  "status": "integer(int64)",
  "totalCents": "computed_field"
}
```

- `id`: Unique identifier for the order (auto-generated if not provided)
- `customerEmail`: Customer's email address (must contain @)
- `lineItems`: Array of items/discounts on the order (minimum 1 required)
- `status`: Current order status (0=pending, 1=charged, 2=fulfilled, 3=cancelled)
- `totalCents`: Computed field (sum of priceCents × quantity for all line items)

### ErrorResponse

Standard error response format.

```json
{
  "code": "string",
  "message": "string"
}
```

### Error Codes

- `order_not_found`: Order does not exist
- `order_already_exists`: Order with this ID already exists
- `invalid_email`: Customer email format is invalid
- `invalid_line_items`: Order must have at least one line item
- `invalid_total`: Order total cannot be negative
- `invalid_status`: Invalid status parameter value
- `order_not_eligible`: Order is not eligible for the requested operation
- `invalid_json`: Request body is not valid JSON
- `internal_error`: Internal server error
- `charge_service_error`: External charge service error

---

## API Endpoints

### Health Check

#### GET /healthz

Check if the service is healthy and running.

**Response Codes:**
- `200 OK`: Service is healthy

**Response:**
```
HTTP 200 OK
(Empty body)
```

---

### Orders

#### GET /orders

Retrieve all orders, optionally filtered by status.

**Query Parameters:**
- `status` (optional): Filter orders by status
  - `pending`: Only pending orders
  - `charged`: Only charged orders  
  - `fulfilled`: Only fulfilled orders
  - `cancelled`: Only cancelled orders
  - (no value): Return all orders

**Example Requests:**
```
GET /orders
GET /orders?status=pending
GET /orders?status=charged
```

**Success Response (200 OK):**
```json
{
  "orders": [
    {
      "id": "2bdd69fa5419e8ed62f7014943b63fa7",
      "customerEmail": "test@example.com",
      "lineItems": [
        {
          "description": "Widget",
          "priceCents": 1000,
          "quantity": 2
        }
      ],
      "status": 0
    }
  ]
}
```

**Error Responses:**
- `400 Bad Request`: Invalid status parameter
  ```json
  {
    "code": "invalid_status",
    "message": "unknown value for status: invalid_status"
  }
  ```
- `500 Internal Server Error`: Storage error
 etc...
  ```json
  {
    "code": "internal_error", 
    "message": "error getting orders: database connection failed"
  }
  ```

#### POST /orders

Create a new order.

**Request Body:**
```json
{
  "customerEmail": "customer@example.com",
  "lineItems": [
    {
      "description": "Product Name",
      "priceCents": 2500,
      "quantity": 1
    }
  ]
}
```

**Validation Rules:**
- `customerEmail`: Required, must contain "@"
- `lineItems`: Required array with at least one item
- Total order amount cannot be negative (sum of priceCents × quantity)

**Success Response (201 Created):**
```json
{
  "order": {
    "id": "generated-order-id",
    "customerEmail": "customer@example.com", 
    "lineItems": [
      {
        "description": "Product Name",
        "priceCents": 2500,
        "quantity": 1
      }
    ],
    "status": 0
  }
}
```

**Error Responses:**
- `400 Bad Request`: Validation errors
  ```json
  {
    "code": "invalid_json",
    "message": "error decoding body: invalid json"
  }
  ```
  ```json
  {
    "code": "invalid_email", 
    "message": "invalid customerEmail"
  }
  ```
  ```json
  {
    "code": "invalid_line_items",
    "message": "an order must contain at least one line item"
  }
  ```
  ```json
  {
    "code": "invalid_total",
    "message": "an order's total cannot be less than 0"
  }
  ```
- `409 Conflict`: Order already exists (when providing custom ID)
  ```json
  {
    "code": "order_already_exists",
    "message": "order already exists"
  }
  ```
- `500 Internal Server Error`: Storage error
  ```json
  {
    "code": "internal_error",
    "message": "error inserting order: database error"
  }
  ```

#### GET /orders/{id}

Retrieve a specific order by ID.

**Path Parameters:**
- `id`: Order identifier

**Success Response (200 OK):**
```json
{
  "order": {
    "id": "12345",
    "customerEmail": "customer@example.com",
    "lineItems": [
      {
        "description": "Product",
        "priceCents": 1000,
        "quantity": 1
      }
    ],
    "status": 1
  }
}
```

**Error Responses:**
- `404 Not Found`: Order does not exist
  ```json
  {
    "code": "order_not_found",
    "message": "not found"
  }
  ```
- `500 Internal Server Error`: Storage error

#### POST /orders/{id}/charge

Charge order payment.

**Path Parameters:**
- `id`: Order identifier

**Request Body:**
```json
{
  "cardToken": "tok_visa_1234"
}
```

**Validation Rules:**
- Order must be in `pending` status (0)
- Order must have positive total amount
- `cardToken`: Required payment token

**Success Response (200 OK):**
```json
{
  "chargedCents": 2500
}
```

**Error Responses:**
- `400 Bad Request`: Invalid JSON or other validation errors
  ```json
  {
    "code": "invalid_json",
    "message": "error decoding body: invalid json"
  }
  ```
- `404 Not Found`: Order does not exist
  ```json
  {
    "code": "order_not_found", 
    "message": "not found"
  }
  ```
- `409 Conflict`: Order not eligible for charging
  ```json
  {
    "code": "order_not_eligible",
    "message": "order ineligible for charging"
  }
  ```
- `500 Internal Server Error`: Charge service or storage errors
  ```json
  {
    "code": "charge_service_error",
    "message": "payment gateway timeout"
  }
  ```
  ```json
  {
    "code": "internal_error",
    "message": "error updating order to charged: database error"
  }
  ```

#### POST /orders/{id}/cancel

Cancel an order.

**Path Parameters:**
- `id`: Order identifier

**Cancellation Rules:**
- Orders can only be cancelled if they are `pending` (0) or `charged` (1)
- `fulfilled` orders cannot be cancelled
- If the order is `charged`, a refund will be processed automatically

**Success Response (200 OK):**

For pending orders:
```json
{
  "message": "order cancelled successfully",
  "orderId": "12345"
}
```

For charged orders (with refund):
```json
{
  "message": "order cancelled successfully", 
  "orderId": "12345",
  "refundedCents": 2500
}
```

**Error Responses:**
- `404 Not Found`: Order does not exist
  ```json
  {
    "code": "order_not_found",
    "message": "not found"
  }
  ```
- `409 Conflict`: Order not eligible for cancellation
  ```json
  {
    "code": "order_not_eligible",
    "message": "order cannot be cancelled - only pending or charged orders can be cancelled"
  }
  ```
- `500 Internal Server Error`: Refund processing or storage errors
  ```json
  {
    "code": "charge_service_error",
    "message": "error processing refund: payment gateway error"
  }
  ```
  ```json
  {
    "code": "internal_error",
    "message": "error cancelling order: database error"
  }
  ```

---

## Order Lifecycle

```
pending (0) → charged (1) → fulfilled (2)
    ↓              ↓
cancelled (3) ← cancelled (3)
```

**Transitions:**
- `pending` → `charged`: Via POST /orders/{id}/charge
- `pending` → `cancelled`: Via POST /orders/{id}/cancel  
- `charged` → `cancelled`: Via POST /orders/{id}/cancel (includes refund)
- `charged` → `fulfilled`: Via external fulfillment system (not documented here)

**Business Rules:**
- Only `pending` orders can be charged
- Only `pending` or `forced` orders can be cancelled
- `fulfilled` orders cannot be cancelled (already shipped)
- Charging a `charged` order returns conflict error

---

## Structured Logging

All API requests are logged with structured data including:

- Request method, path, and status code
- Request duration in milliseconds
- Client IP and User-Agent
- Order ID (when applicable)
- Handler-specific context (order counts, status filters, etc.)
- Error details for failed requests

Example log entries:
```
INFO: get orders request started handler=getOrders
INFO: fetching orders from storage handler=getOrders status_filter=pending status_code=0
INFO: successfully retrieved orders from storage handler=getOrders order_count=5
INFO: request completed successfully method=GET path=/orders status_code=200 duration_ms=15 client_ip=127.0.0.1
```
