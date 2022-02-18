# order-up

Code candidates will extend as part of their technical interview process. The
order-up service handles all order-specific calls including creating orders,
checking the status on orders, etc. This service is part of a larger microservice
backend for a online marketplace.

## Getting started

You also will need to [install Go](https://go.dev/doc/install). Then clone this
this repository and run `go mod tidy` within this repository to download all
necessary dependencies locally.

## Project Structure

### Top-level

The top-level only contains a single `main.go` file which holds the `main`
function. If you ran `go build ./.` that would produce a `order-up` binary that
would start by executing the `main` function in `main.go`.

### api package

The `api` package handles incoming HTTP requests with a REST paradigm and calls
various functions based on the path. This package uses the `storage` package to
perform the necessary functionality for each API call. The tests use a mocked
storage instance.

### storage package

The `storage` package contains an in-memory implementation for persisting and retrieving orders. You are expected to extend this implementation to satisfy the tests and documented functionality.

### mocks package

The `mocks` package just contains a helper function for mocking an external
service by accepting an http.Handler and returning a *http.Client as well as
generated code for mocking a `*storage.Instance`. This simply makes the tests
easier in the `api` package.

## Relevant Go commands

* [`go mod tidy`](https://go.dev/ref/mod#go-mod-tidy) downloads all dependencies
and update `go.mod` file with any new dependencies
* [`go test -v -race ./...`](https://pkg.go.dev/cmd/go#hdr-Test_packages) tests all
files and subdirectories. You can instead do `go test -v ./storage/...` to only
test the storage package. Any public function with the format `TestX(*testing.T)`
will automatically be called by `go test`. Typically these functions are placed
in `X_test.go` files. You can pass a regex to `-run` like `-run ^TestInsertOrder$`
in order to just run tests matching the regex.
* [`go fmt ./...`](https://pkg.go.dev/cmd/go#hdr-Gofmt__reformat__package_sources)
reformats the go files according to the gofmt spec
* [`go vet ./...`](https://pkg.go.dev/cmd/go#hdr-Report_likely_mistakes_in_packages)
prints out most-likely errors or mistakes in Go code
* [`go get $package`](https://pkg.go.dev/cmd/go#hdr-Add_dependencies_to_current_module_and_install_them)
adds a new dependency to the current project


