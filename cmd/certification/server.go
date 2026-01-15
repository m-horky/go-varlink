package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"os"
	"reflect"

	govarlink "github.com/emersion/go-varlink"
	"github.com/emersion/go-varlink/internal/certification"
)

// clientState keeps track of values the client has to return.
type clientState struct {
	bool      bool
	integer   int
	float     float64
	string    string
	dtoStruct struct {
		Bool   bool    `json:"bool"`
		Float  float64 `json:"float"`
		Int    int     `json:"int"`
		String string  `json:"string"`
	}
	dtoMap    map[string]string
	dtoSet    map[string]struct{}
	dtoMyType certification.MyType
}

func newState() *clientState {
	result := &clientState{}
	result.dtoMap = make(map[string]string)
	result.dtoSet = make(map[string]struct{})
	return result
}

type certificationBackend struct {
	clientStates map[string]*clientState
}

// isEqual compares 'got' and 'wants' values, returning a CertificationErrorError if they differ.
func isEqual(got, wants any) error {
	if reflect.DeepEqual(got, wants) {
		return nil
	}
	// Marshal both to JSON for comparison
	gotJSON, err := json.Marshal(got)
	if err != nil {
		return err
	}
	wantsJSON, err := json.Marshal(wants)
	if err != nil {
		return err
	}
	// Compare JSON representations (this handles map ordering and StringSet differences)
	if string(gotJSON) == string(wantsJSON) {
		return nil
	}
	return &certification.CertificationErrorError{
		Got:   gotJSON,
		Wants: wantsJSON,
	}
}

// randomString generates a random string of the specified length.
func randomString(length int) string {
	runes := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	result := make([]rune, length)
	for i := range result {
		result[i] = runes[rand.Intn(len(runes))]
	}
	return string(result)
}

// newClient generates an ID and initiates its 'clientState' state.
func (c *certificationBackend) newClient() string {
	clientId := randomString(32)
	c.clientStates[clientId] = newState()
	return clientId
}

func (c *certificationBackend) Start(_ *certification.StartIn) (*certification.StartOut, error) {
	clientId := c.newClient()
	log.Printf("New client: %s\n", clientId)
	return &certification.StartOut{ClientId: clientId}, nil
}

func (c *certificationBackend) Test01(in *certification.Test01In) (*certification.Test01Out, error) {
	log.Printf("Test01: Received=%s\n", in.ClientId)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}

	value := rand.Int()%2 == 0
	state.bool = value
	log.Printf("Test01: Sent=%t\n", value)
	return &certification.Test01Out{Bool: value}, nil
}

func (c *certificationBackend) Test02(in *certification.Test02In) (*certification.Test02Out, error) {
	log.Printf("Test02: Received=%t\n", in.Bool)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.Bool, state.bool); err != nil {
		return nil, err
	}

	value := rand.Int()
	state.integer = value
	log.Printf("Test02: Sent=%d\n", value)
	return &certification.Test02Out{Int: value}, nil
}

func (c *certificationBackend) Test03(in *certification.Test03In) (*certification.Test03Out, error) {
	log.Printf("Test03: Received=%v\n", in.Int)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.Int, state.integer); err != nil {
		return nil, err
	}

	value := rand.Float64()
	state.float = value
	log.Printf("Test03: Sent=%f\n", value)
	return &certification.Test03Out{Float: value}, nil
}

func (c *certificationBackend) Test04(in *certification.Test04In) (*certification.Test04Out, error) {
	log.Printf("Test04: Received=%v\n", in.Float)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.Float, state.float); err != nil {
		return nil, err
	}

	value := randomString(8)
	state.string = value
	log.Printf("Test04: Sent=%s\n", value)
	return &certification.Test04Out{String: value}, nil
}

func (c *certificationBackend) Test05(in *certification.Test05In) (*certification.Test05Out, error) {
	log.Printf("Test05: Received=%s\n", in.String)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.String, state.string); err != nil {
		return nil, err
	}

	log.Printf("Test05: Sent=%t,%d,%f,%s}\n",
		state.bool, state.integer, state.float, state.string)
	return &certification.Test05Out{
		Bool:   state.bool,
		Int:    state.integer,
		Float:  state.float,
		String: state.string,
	}, nil
}

func (c *certificationBackend) Test06(in *certification.Test06In) (*certification.Test06Out, error) {
	log.Printf("Test06: Received=%t,%d,%f,%s}\n",
		in.Bool, in.Int, in.Float, in.String)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.Bool, state.bool); err != nil {
		return nil, err
	}
	if err := isEqual(in.Int, state.integer); err != nil {
		return nil, err
	}
	if err := isEqual(in.Float, state.float); err != nil {
		return nil, err
	}
	if err := isEqual(in.String, state.string); err != nil {
		return nil, err
	}

	state.dtoStruct.Bool = in.Bool
	state.dtoStruct.Int = in.Int
	state.dtoStruct.Float = in.Float
	state.dtoStruct.String = in.String
	out := &certification.Test06Out{}
	out.Struct = state.dtoStruct
	log.Printf("Test06: Sent %v\n", out.Struct)
	return out, nil
}

func (c *certificationBackend) Test07(in *certification.Test07In) (*certification.Test07Out, error) {
	log.Printf("Test07: Received %v\n", in.Struct)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.Struct, state.dtoStruct); err != nil {
		return nil, err
	}

	state.dtoMap = map[string]string{"key1": randomString(8), "key2": randomString(8)}
	log.Printf("Test07: Sent %v\n", state.dtoMap)
	return &certification.Test07Out{Map: state.dtoMap}, nil
}

func (c *certificationBackend) Test08(in *certification.Test08In) (*certification.Test08Out, error) {
	log.Printf("Test08: Received %v\n", in.Map)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.Map, state.dtoMap); err != nil {
		return nil, err
	}

	for k := range state.dtoMap {
		state.dtoSet[k] = struct{}{}
	}
	log.Printf("Test08: Sent %#v\n", state.dtoSet)
	return &certification.Test08Out{Set: state.dtoSet}, nil
}

func (c *certificationBackend) Test09(in *certification.Test09In) (*certification.Test09Out, error) {
	log.Printf("Test09: Received %v\n", in.Set)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.Set, state.dtoSet); err != nil {
		return nil, err
	}

	myType := certification.MyType{
		Object:     json.RawMessage(`{"foo":"bar"}`),
		Enum:       "two",
		Array:      []string{"one", "two", "three"},
		Dictionary: map[string]string{"key1": "value1", "key2": "value2"},
		Stringset:  map[string]struct{}{"set1": {}, "set2": {}},
		Struct: struct {
			First  int    `json:"first"`
			Second string `json:"second"`
		}{
			First:  state.integer,
			Second: state.string,
		},
	}
	nullable := "nullable-value"
	myType.Nullable = &nullable
	state.dtoMyType = myType
	log.Printf("Test09: Sent %v\n", myType)
	return &certification.Test09Out{Mytype: myType}, nil
}

func (c *certificationBackend) Test10(in *certification.Test10In) (*certification.Test10Out, error) {
	log.Printf("Test10: Received %v\n", in.Mytype)
	state, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}
	if err := isEqual(in.Mytype, state.dtoMyType); err != nil {
		return nil, err
	}

	// FIXME Add streaming
	reply := "test10-reply"
	log.Printf("Test10: Sent %q\n", reply)
	return &certification.Test10Out{String: reply}, nil
}

func (c *certificationBackend) Test11(in *certification.Test11In) (*certification.Test11Out, error) {
	log.Printf("Test11: Received %v\n", in.LastMoreReplies)
	_, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}

	// FIXME Is this how oneway messages are handled?
	log.Printf("Test11: Sent\n")
	return &certification.Test11Out{}, nil
}

func (c *certificationBackend) End(in *certification.EndIn) (*certification.EndOut, error) {
	log.Printf("End: Received\n")
	_, ok := c.clientStates[in.ClientId]
	if !ok {
		return nil, &certification.ClientIdErrorError{}
	}

	delete(c.clientStates, in.ClientId)
	log.Printf("End: Sent AllOk=true\n")
	return &certification.EndOut{AllOk: true}, nil
}

func server(protocol, socket string) {
	log.Printf("starting certification server on %s://%s\n", protocol, socket)

	registry := govarlink.NewRegistry(&govarlink.RegistryOptions{
		Vendor:  "emersion",
		Product: "go-varlink certification server",
		Version: "1.0",
		URL:     "https://github.com/emersion/go-varlink",
	})

	backend := &certificationBackend{
		clientStates: make(map[string]*clientState),
	}
	certification.Handler{Backend: backend}.Register(registry)

	if protocol == "unix" {
		_ = os.Remove(socket)
	}

	listener, err := net.Listen(protocol, socket)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	certificationServer := govarlink.NewServer()
	certificationServer.Handler = registry
	if err = certificationServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
