package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/emersion/go-varlink"
	"github.com/emersion/go-varlink/example/internal/varlink/calcapi"
	"github.com/emersion/go-varlink/example/internal/varlink/stringapi"
)

type calcBackend struct{}

func (calcBackend) Multiply(in *calcapi.MultiplyIn) (*calcapi.MultiplyOut, error) {
	return &calcapi.MultiplyOut{Result: in.A * in.B}, nil
}

func (calcBackend) Divide(in *calcapi.DivideIn) (*calcapi.DivideOut, error) {
	if in.B == 0 {
		return nil, &calcapi.DivisionByZeroError{}
	}
	return &calcapi.DivideOut{Result: in.A / in.B}, nil
}

type stringBackend struct{}

func (stringBackend) Repeat(in *stringapi.RepeatIn) (*stringapi.RepeatOut, error) {
	return &stringapi.RepeatOut{Output: in.Input}, nil
}

func (stringBackend) Reverse(in *stringapi.ReverseIn) (*stringapi.ReverseOut, error) {
	result := make([]rune, len(in.Input))
	for i, char := range in.Input {
		result[len(in.Input)-i-1] = char
	}
	return &stringapi.ReverseOut{Output: string(result)}, nil
}

func (stringBackend) Random(_ *stringapi.RandomIn) (*stringapi.RandomOut, error) {
	// chosen by a fair dice roll, guaranteed to be random.
	return &stringapi.RandomOut{Output: "4"}, nil
}

// divideWithRetry calls Divide and retries once on unexpected errors,
// wrapping the error from the first attempt with fmt.Errorf + %w.
// This simulates middleware that adds context to errors — the kind of
// wrapping that breaks a bare type assertion but is handled correctly
// by errors.As in the generated unmarshalError.
func divideWithRetry(c calcapi.Client, a, b int) (*calcapi.DivideOut, error) {
	out, err := c.Divide(&calcapi.DivideIn{A: a, B: b})
	if err != nil {
		// Wrap the error with context, as real middleware often does.
		wrapped := fmt.Errorf("attempt 1 of divide(%d,%d): %w", a, b, err)

		// Without errors.As in unmarshalError this check would always fail,
		// because the concrete type is now hidden inside *fmt.wrapError.
		var divByZero *calcapi.DivisionByZeroError
		if errors.As(wrapped, &divByZero) {
			// The error is definitively DivisionByZero — no point retrying.
			return nil, wrapped
		}

		// Some other error: retry once.
		out, err = c.Divide(&calcapi.DivideIn{A: a, B: b})
		if err != nil {
			return nil, fmt.Errorf("attempt 2 of divide(%d,%d): %w", a, b, err)
		}
	}
	return out, nil
}

func main() {
	registry := varlink.NewRegistry(&varlink.RegistryOptions{
		Vendor:  "emersion/go-varlink",
		Product: "usage example",
		Version: "1.0",
		URL:     "https://github.com/emersion/go-varlink",
	})

	calcapi.Handler{Backend: calcBackend{}}.Register(registry)
	stringapi.Handler{Backend: stringBackend{}}.Register(registry)

	_ = os.Remove("./org.example.sock")
	listener, err := net.Listen("unix", "./org.example.sock")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer listener.Close()

	server := varlink.NewServer()
	server.Handler = registry

	// Run the server in the background so we can make a client call below.
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err.Error())
		}
	}()

	// Connect a client and exercise divideWithRetry to demonstrate that
	// errors.As correctly unwraps a *ClientError even after it has been
	// wrapped with fmt.Errorf("...: %w", err).
	conn, err := net.Dial("unix", "./org.example.sock")
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	rawClient := varlink.NewClient(conn)
	defer rawClient.Close()

	c := calcapi.Client{Client: rawClient}

	// Normal division — should succeed.
	out, err := divideWithRetry(c, 12, 3)
	if err != nil {
		log.Fatalf("12/3 unexpected error: %v", err)
	}
	fmt.Printf("12 / 3 = %d\n", out.Result)

	// Division by zero — divideWithRetry wraps the error with fmt.Errorf+%w,
	// then checks errors.As for *DivisionByZeroError. Without the errors.As
	// fix in the generated unmarshalError the typed error would never be
	// returned, errors.As would fail here, and the retry would fire
	// unnecessarily — giving a misleading "attempt 2" error instead.
	_, err = divideWithRetry(c, 7, 0)
	var divByZero *calcapi.DivisionByZeroError
	if errors.As(err, &divByZero) {
		fmt.Println("7 / 0: correctly identified as DivisionByZero (no retry attempted)")
	} else {
		log.Fatalf("7 / 0: expected DivisionByZeroError, got %T: %v", err, err)
	}
}
