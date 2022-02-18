// Package main is the first package loaded when running the service. It
// instantiates any necessary functionality from other packages.
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/levenlabs/order-up/api"
	"github.com/levenlabs/order-up/mocks"
	"github.com/levenlabs/order-up/storage"
)

func main() {
	// flag.String returns a pointer to a string value that is set after
	// flag.Parse() is called
	addr := flag.String("listen-addr", "localhost:8888", "the address to listen on for API requests")
	flag.Parse()

	server := new(http.Server)
	// we dereference the address flag and set it on the server so the
	// ListenAndServe call later knows what address to Listen on
	server.Addr = *addr
	// here we're calling the api package's Handler() function to get an instance of
	// an http.Handler that we can set as the server's Handler
	// on every HTTP request the server will call the handler's ServeHTTP function
	server.Handler = api.Handler(
		storage.NewMemory(),
		// we would replace these with actual clients that talk to the underlying services
		// but for this contrived service we just iuggno
		mocks.NewMockedService(unimplementedHandler),
		mocks.NewMockedService(unimplementedHandler),
	)

	// if we just called ListenAndServe directly then we would never return since
	// ListenAndServe starts listening for HTTP requests and blocks until the
	// server is shutdown
	go server.ListenAndServe()
	// we want to gracefully shutdown the server right before the process stops
	// the context isn't that important for this service but you could call something
	// like context.WithTimeout if you wanted to only give the HTTP server a limited
	// amount of time to shutdown
	defer server.Shutdown(context.Background())

	// if main returns then the process stops running so we instead wait for an
	// interrupt signal (Ctrl+C) by creating a channel, passing it to the signal
	// package and then waiting to receive something from the channel
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	// once we receive something over this channel we will continue the function
	// and end up returning, causing the process to stop
	<-ch
}

var unimplementedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
})
