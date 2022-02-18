## Order-Up Service Coding Challenge

Welcome to the Order-Up Service coding challenge! This is a 3-hour, real-time session where you'll work on a Go-based microservice. The service uses an in-memory database, allowing you to focus on business logic, API design, and code quality.

### Your Task: Choose Your Challenges

Below is a menu of potential challenges, categorized by the skills they test. **You are not expected to complete them all.** Only as much as you reasonably can in 3 hours.

Your goal is to choose the tasks that you feel best showcase your skills and make as much progress as you can in the three-hour window. We are more interested in your thought process, the quality of your work, and how you approach problems than in the number of tasks you complete.

AI tools are allowed and encouraged.

Please be prepared to discuss why you chose the tasks you did, and to speak in depth about your approaches and solutions. You should be able to demonstrate a deep understanding of any solutions written by an LLM.

--- 

### The Challenge Menu

#### Core Feature Development

These tasks involve building new, user-facing functionality.

| Task | Description | What This Signals | Difficulty |
| :--- | :--- | :--- | :--- |
| **Implement Order Cancellation** | Implement a `POST /orders/:id/cancel` endpoint. This includes adding a new `OrderStatusCanceled`, handling refunds for charged orders via `innerChargeOrder`, and writing tests. | Core backend logic, API implementation, testing, understanding business rules. | Medium |
| **Implement Order Fulfillment** | Implement a `POST /orders/:id/fulfill` endpoint. This should call the (mocked) `fulfillmentService` for each line item and then update the order's status to `OrderStatusFulfilled`. | Ability to work with external services, handle loops and potential partial failures, API implementation. | Medium |
| **Allow Order Modification** | Implement a `PUT /orders/:id` endpoint that allows changing the `LineItems` on a pending order. You'll need to decide how to handle changes to the total cost and ensure the order total doesn't become negative. | Deeper data manipulation logic, handling edge cases, API design (how to represent the change). | Large |

#### API & Developer Experience (DX) Improvements

These tasks focus on making the API more robust and easier for other developers to use.

| Task | Description | What This Signals | Difficulty |
| :--- | :--- | :--- | :--- |
| **Add Pagination to `GET /orders`** | The current `GET /orders` endpoint returns all orders at once. Modify it to accept `limit` and `offset` query parameters to allow for pagination. | Understanding of scalable API design, storage querying, and clear API response structure. | Medium |
| **Create Structured Error Responses** | Currently, errors are returned as a simple JSON string: `{"error": "..."}`. Refactor the API handlers to return a structured error object, e.g., `{"code": "order_not_found", "message": "The requested order does not exist."}`. | Attention to detail, empathy for API consumers, code refactoring skills. | Small |
| **Add API Documentation** | Create a new file, `docs/api.md`, and document the available API endpoints. For each endpoint, describe its purpose, parameters, and possible success and error responses using a standard format like OpenAPI. | Communication skills, ability to write clear technical documentation, thinking from a user's perspective. | Small |

#### Code Quality & Production Readiness

These tasks involve improving the existing codebase for maintainability and observability.

| Task | Description | What This Signals | Difficulty |
| :--- | :--- | :--- | :--- |
| **Add Structured Logging** | The service currently has no logging. Use the `llog` package (already a dependency) to add structured logs to the API handlers. Log key information like the order ID, the request path, and the time taken to process the request. | Understanding of observability, production-readiness, and what information is critical for debugging. | Small |
| **Refactor Handler Logic** | The API handlers in `api/api.go` have duplicated code for fetching an order and handling "not found" errors. Create a helper function or a Gin middleware to centralize this logic and improve readability. | Code design and abstraction skills, ability to identify and reduce code smells (DRY principle). | Medium |
| **Add a Health Check Endpoint** | Add a `GET /healthz` endpoint that returns a `200 OK` status. In a real system, this would check dependencies (like a database), but for now, just returning `200` is sufficient. This is a standard for production services. | Knowledge of operational best practices and production readiness for microservices. | Small |

#### "Hard Mode" Challenges

These tasks are intentionally ambiguous or complex, requiring deep system understanding, debugging, or architectural reasoning.

| Task | Description | What This Signals | Difficulty |
| :--- | :--- | :--- | :--- |
| **Fix the Race Condition** | The `storage/memory.go` implementation has a subtle race condition. Use `go test -race` to find it, then explain *why* it happens and fix it. (Hint: The bug is not in the provided `MemoryInstance` but in how it might be used or extended). | **Debugging skills, deep understanding of concurrency.** AI can explain race conditions, but struggles to find them in code and reason about their cause. | Large |
| **Diagnose Performance Issues** | The `GetOrders` function is slow when there are many orders. Your task is not just to fix it, but to first *prove* it's slow. Use Go's profiling tools (`pprof`) to identify the bottleneck. Then, propose and implement a fix. | **Analytical and diagnostic skills.** This tests the scientific method of engineering: hypothesize, measure, and then act. | Hard |
--- 

### Getting Started

1.  Clone the repository.
2.  Run `go mod tidy` to download dependencies.
3.  Run `go test -v -race ./...` to run the existing tests. Use the tests as a guide to understand the existing functionality.

You have 3 hours. Good luck!
