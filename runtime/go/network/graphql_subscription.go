// GraphQL subscription support over WebSocket (graphql-ws protocol).
package network

import (
	"fmt"
	"net/http"
	"reflect"

	graphql "github.com/hasura/go-graphql-client"
)

// Subscription is a live GraphQL subscription. Updates streams one GraphQLResult per
// server message (Response holds a freshly decoded copy of the subscription struct,
// or Error is set). Stop ends the subscription and closes Updates.
type Subscription struct {
	updates chan GraphQLResult
	client  *graphql.SubscriptionClient
}

// Updates returns the channel of subscription results. It is closed when the
// subscription stops (via Stop, a fatal error, or server completion).
func (s *Subscription) Updates() <-chan GraphQLResult { return s.updates }

// Stop ends the subscription and releases the underlying WebSocket connection.
func (s *Subscription) Stop() error { return s.client.Close() }

// Subscribe opens a GraphQL subscription. subscription is a struct whose graphql
// tags define the shape; variables may be nil. Each server message is decoded into a
// new value of the subscription's type and delivered on the returned Subscription's
// Updates channel. The ws/wss endpoint is derived from the client's URL, and
// ConnectionOptions.Headers are sent on the WebSocket handshake.
func (g *GraphQLClient) Subscribe(subscription interface{}, variables map[string]interface{}) (*Subscription, error) {
	wsOpts := websocketURL(g.URL)
	fullURL, err := buildFullURL(wsOpts, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to build subscription URL: %w", err)
	}

	subClient := graphql.NewSubscriptionClient(fullURL)
	if len(g.Headers) > 0 {
		header := http.Header{}
		for k, v := range g.Headers {
			header.Set(k, v)
		}
		subClient = subClient.WithWebSocketOptions(graphql.WebsocketOptions{HTTPHeader: header})
	}

	sub := &Subscription{
		updates: make(chan GraphQLResult, 1),
		client:  subClient,
	}

	// elemType is the (non-pointer) type of the subscription struct, used to decode
	// each message into a fresh typed value.
	elemType := reflect.TypeOf(subscription)
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if _, err := subClient.Subscribe(subscription, variables, sub.handler(elemType)); err != nil {
		_ = subClient.Close()
		return nil, fmt.Errorf("failed to start subscription: %w", err)
	}

	go func() {
		defer close(sub.updates)
		if runErr := subClient.Run(); runErr != nil {
			sub.updates <- GraphQLResult{Error: fmt.Errorf("subscription stopped: %w", runErr)}
		}
	}()

	return sub, nil
}

// handler returns a message callback that decodes each payload into a new value of
// elemType and forwards it (or the error) onto the updates channel.
func (s *Subscription) handler(elemType reflect.Type) func([]byte, error) error {
	return func(message []byte, err error) error {
		if err != nil {
			s.updates <- GraphQLResult{Error: err}
			return nil
		}
		out := reflect.New(elemType)
		if decodeErr := graphql.UnmarshalGraphQL(message, out.Interface()); decodeErr != nil {
			s.updates <- GraphQLResult{Error: fmt.Errorf("failed to decode subscription message: %w", decodeErr)}
			return nil
		}
		s.updates <- GraphQLResult{Response: out.Interface()}
		return nil
	}
}
