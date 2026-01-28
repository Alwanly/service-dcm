package pubsub

import "context"

// Message represents a pub/sub message
type Message struct {
	Channel string
	Payload string
}

// Publisher defines the interface for publishing messages
type Publisher interface {
	// Publish publishes a message to a channel
	Publish(ctx context.Context, channel string, message string) error
	Close() error
}

// Subscriber defines the interface for subscribing to messages
type Subscriber interface {
	// Subscribe subscribes to one or more channels and returns a message channel
	Subscribe(ctx context.Context, channels ...string) (<-chan Message, error)
	Unsubscribe(ctx context.Context, channels ...string) error
	Close() error
}

// PubSub combines Publisher and Subscriber
type PubSub interface {
	Publisher
	Subscriber
}
