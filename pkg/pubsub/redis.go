package pubsub

import (
	"context"
	"fmt"

	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type redisPubSub struct {
	client    *redis.Client
	pubsub    *redis.PubSub
	logger    *logger.CanonicalLogger
	messageCh chan Message
	cancel    context.CancelFunc
}

func NewRedisPubSub(cfg RedisConfig, log *logger.CanonicalLogger) (PubSub, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Try a ping to validate connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis at %s: %w", addr, err)
	}

	r := &redisPubSub{
		client:    client,
		logger:    log,
		messageCh: make(chan Message, 16),
	}

	log.Info("redis client initialized", logger.String("addr", addr))

	return r, nil
}

// Publish publishes a message to a Redis channel
func (r *redisPubSub) Publish(ctx context.Context, channel string, message string) error {
	if err := r.client.Publish(ctx, channel, message).Err(); err != nil {
		r.logger.WithError(err).Error("failed to publish message to redis")
		return err
	}
	return nil
}

// Ping checks if Redis connection is healthy
func (r *redisPubSub) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// IsHealthy returns true if Redis connection is active
func (r *redisPubSub) IsHealthy(ctx context.Context) bool {
	return r.Ping(ctx) == nil
}

// Subscribe subscribes to Redis channels
func (r *redisPubSub) Subscribe(ctx context.Context, channels ...string) (<-chan Message, error) {
	if len(channels) == 0 {
		return nil, nil
	}

	r.pubsub = r.client.Subscribe(ctx, channels...)

	// Start listening
	listenCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go r.listen(listenCtx)

	r.logger.Info("subscribed to redis channels", logger.Any("channels", channels))
	return r.messageCh, nil
}

// Unsubscribe unsubscribes from Redis channels
func (r *redisPubSub) Unsubscribe(ctx context.Context, channels ...string) error {
	if r.pubsub == nil {
		return nil
	}
	return r.pubsub.Unsubscribe(ctx, channels...)
}

// Close closes the Redis connection
func (r *redisPubSub) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	if r.pubsub != nil {
		_ = r.pubsub.Close()
	}
	if r.client != nil {
		if err := r.client.Close(); err != nil {
			r.logger.WithError(err).Error("failed to close redis client")
			return err
		}
	}
	close(r.messageCh)
	return nil
}

// listen listens for messages from subscribed channels
func (r *redisPubSub) listen(ctx context.Context) {
	ch := r.pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopping redis listener")
			return
		case m, ok := <-ch:
			if !ok {
				r.logger.Info("redis pubsub channel closed")
				return
			}
			r.messageCh <- Message{Channel: m.Channel, Payload: m.Payload}
		}
	}
}
