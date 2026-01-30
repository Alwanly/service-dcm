package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/dto"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/poll"
	"github.com/Alwanly/service-distribute-management/pkg/pubsub"
	"go.uber.org/zap"
)

type StoreData struct {
	Config       *models.Configuration
	ETag         string
	AgentID      string
	PollURL      string
	PollInterval int
	APIToken     string
}

type Repository struct {
	store         *StoreData
	storeMutex    sync.RWMutex
	pubsub        pubsub.Subscriber
	configPoller  poll.Poller
	agentID       string
	controllerURL string
	workerURL     string
	apiToken      string
	// Redis circuit breaker fields
	redisFailures    int
	redisCircuitOpen bool
	lastRedisFailure time.Time
	circuitMutex     sync.Mutex
}

// NewRepository creates a new repository instance
func NewRepository(controllerURL string, workerURL string, agentID string, apiToken string, subscriber pubsub.Subscriber) IRepository {
	return &Repository{
		store:         &StoreData{},
		storeMutex:    sync.RWMutex{},
		pubsub:        subscriber,
		configPoller:  nil,
		agentID:       agentID,
		controllerURL: controllerURL,
		workerURL:     workerURL,
		apiToken:      apiToken,
	}
}

// SetAPIToken stores the API token for future requests
func (r *Repository) SetAPIToken(token string) {
	r.storeMutex.Lock()
	defer r.storeMutex.Unlock()
	if r.store == nil {
		r.store = &StoreData{}
	}
	r.store.APIToken = token
	r.apiToken = token
}

// GetAPIToken returns the stored API token
func (r *Repository) GetAPIToken() string {
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()
	if r.store == nil {
		return ""
	}
	return r.store.APIToken
}

// SetConfig stores configuration and its ETag
func (r *Repository) SetConfig(config *models.Configuration, etag string) {
	r.storeMutex.Lock()
	defer r.storeMutex.Unlock()
	if r.store == nil {
		r.store = &StoreData{}
	}
	r.store.Config = config
	r.store.ETag = etag
}

// GetConfig retrieves stored configuration and ETag
func (r *Repository) GetConfig() (*models.Configuration, string) {
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()
	if r.store == nil {
		return nil, ""
	}
	return r.store.Config, r.store.ETag
}

// GetPollInfo retrieves the poll URL and interval
func (r *Repository) GetPollInfo() (string, int, error) {
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()
	if r.store == nil {
		return "", 0, nil
	}
	return r.store.PollURL, r.store.PollInterval, nil
}

// SetPollInfo sets the poll URL and interval
func (r *Repository) SetPollInfo(pollURL string, pollInterval int) error {
	r.storeMutex.Lock()
	defer r.storeMutex.Unlock()
	if r.store == nil {
		r.store = &StoreData{}
	}
	r.store.PollURL = pollURL
	r.store.PollInterval = pollInterval
	return nil
}

// UpdatePollInterval updates the stored polling interval
func (r *Repository) UpdatePollInterval(newInterval int) {
	r.storeMutex.Lock()
	defer r.storeMutex.Unlock()
	if r.store == nil {
		r.store = &StoreData{}
	}
	r.store.PollInterval = newInterval
}

// handleConfigUpdate processes configuration updates from either push or poll
func (r *Repository) handleConfigUpdate(ctx context.Context, log *logger.CanonicalLogger, etag string, correlationID string) error {
	updateStart := time.Now()

	r.storeMutex.RLock()
	if r.store != nil && r.store.ETag == etag {
		r.storeMutex.RUnlock()
		log.Debug("Configuration already up to date", zap.String("etag", etag))
		return nil
	}
	r.storeMutex.RUnlock()

	// Fetch configuration from controller
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/config", r.controllerURL), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if r.agentID != "" {
		req.Header.Set("X-Agent-ID", r.agentID)
	}
	if r.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiToken)
	}
	if correlationID != "" {
		req.Header.Set("X-Correlation-ID", correlationID)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch config from controller: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("controller returned status %d", resp.StatusCode)
	}

	var cr dto.ConfigurationResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return fmt.Errorf("failed to decode controller config response: %w", err)
	}

	// Convert to models.Configuration
	cfg := &models.Configuration{
		ID:   cr.ID,
		ETag: cr.ETag,
	}
	if data, err := json.Marshal(cr.Config); err == nil {
		cfg.ConfigData = string(data)
	}

	oldETag := ""
	r.storeMutex.Lock()
	if r.store != nil {
		oldETag = r.store.ETag
	}
	if r.store == nil {
		r.store = &StoreData{}
	}
	r.store.Config = cfg
	r.store.ETag = cr.ETag
	r.storeMutex.Unlock()

	elapsed := time.Since(updateStart)
	log.Info("Configuration updated via notification",
		zap.String("old_etag", oldETag),
		zap.String("new_etag", cr.ETag),
		zap.String("delivery_method", "push"),
		zap.Duration("duration_ms", elapsed),
		zap.String("correlation_id", correlationID),
	)

	// Forward updated config to worker and include correlation id
	if r.workerURL != "" {
		configData := new(models.ConfigData)
		if cfg.ConfigData != "" {
			_ = json.Unmarshal([]byte(cfg.ConfigData), configData)
		}
		payload := dto.SendConfigRequest{ID: cfg.ID, ETag: cfg.ETag, ConfigData: *configData}
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			log.WithError(err).Error("failed to marshal config for worker")
			return nil
		}
		workerReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/config", r.workerURL), bytes.NewReader(bodyBytes))
		if err != nil {
			log.WithError(err).Error("failed to create worker request")
			return nil
		}
		workerReq.Header.Set("Content-Type", "application/json")
		corr := correlationID
		if corr == "" {
			corr = uuid.Must(uuid.NewV7()).String()
		}
		workerReq.Header.Set("X-Correlation-ID", corr)
		if r.apiToken != "" {
			workerReq.Header.Set("Authorization", "Bearer "+r.apiToken)
		}
		client := &http.Client{Timeout: 10 * time.Second}
		wresp, err := client.Do(workerReq)
		if err != nil {
			log.WithError(err).Error("failed to send config to worker")
			return nil
		}
		wresp.Body.Close()
		if wresp.StatusCode != http.StatusOK {
			log.Error("worker rejected config", zap.Int("status", wresp.StatusCode))
			return nil
		}
		log.Info("configuration forwarded to worker via push", zap.String("etag", cfg.ETag), zap.String("correlation_id", corr))
	}

	return nil
}

// RegisterConfigPolling starts periodic fallback configuration polling
func (r *Repository) RegisterConfigPolling(ctx context.Context, log *logger.CanonicalLogger) {
	if r == nil {
		return
	}

	// Start a fallback poller that performs conditional GETs against the controller
	go func() {
		// determine initial interval
		interval := 60 * time.Second
		r.storeMutex.RLock()
		if r.store != nil && r.store.PollInterval > 0 {
			interval = time.Duration(r.store.PollInterval) * time.Second
		}
		r.storeMutex.RUnlock()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Info("config fallback polling started", zap.Duration("interval", interval))

		client := &http.Client{Timeout: 15 * time.Second}

		for {
			select {
			case <-ctx.Done():
				log.Info("config fallback polling stopped")
				return
			case <-ticker.C:
				// read current ETag and poll URL
				r.storeMutex.RLock()
				curETag := ""
				pollURL := r.store.PollURL
				agentID := r.agentID
				token := r.apiToken
				if r.store != nil {
					curETag = r.store.ETag
				}
				r.storeMutex.RUnlock()

				target := fmt.Sprintf("%s/config", r.controllerURL)
				if pollURL != "" {
					// if controller provided an explicit poll URL, use it
					target = fmt.Sprintf("%s%s", r.controllerURL, pollURL)
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
				if err != nil {
					log.WithError(err).Error("failed to create poll request")
					continue
				}
				if curETag != "" {
					req.Header.Set("If-None-Match", curETag)
				}
				if agentID != "" {
					req.Header.Set("X-Agent-ID", agentID)
				}
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}

				resp, err := client.Do(req)
				if err != nil {
					log.WithError(err).Error("poll request failed")
					continue
				}

				if resp.StatusCode == http.StatusNotModified {
					resp.Body.Close()
					// nothing to do
					continue
				}
				if resp.StatusCode != http.StatusOK {
					log.Error("poll returned non-OK status", zap.Int("status", resp.StatusCode))
					resp.Body.Close()
					continue
				}

				// decode body separately to avoid holding locks while reading
				var cr dto.ConfigurationResponse
				if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
					resp.Body.Close()
					log.WithError(err).Error("failed to decode config response from poll")
					continue
				}
				resp.Body.Close()

				// update store with new config and forward to worker
				cfg := &models.Configuration{ID: cr.ID, ETag: cr.ETag}
				if data, err := json.Marshal(cr.Config); err == nil {
					cfg.ConfigData = string(data)
				}

				// store update
				r.storeMutex.Lock()
				oldETag := ""
				if r.store != nil {
					oldETag = r.store.ETag
				}
				if r.store == nil {
					r.store = &StoreData{}
				}
				r.store.Config = cfg
				r.store.ETag = cr.ETag
				r.storeMutex.Unlock()

				log.Info("Configuration updated via poll",
					zap.String("old_etag", oldETag),
					zap.String("new_etag", cr.ETag),
					zap.String("delivery_method", "poll"),
				)

				// forward to worker
				if r.workerURL != "" {
					// prepare payload
					configData := new(models.ConfigData)
					if cfg.ConfigData != "" {
						_ = json.Unmarshal([]byte(cfg.ConfigData), configData)
					}
					payload := dto.SendConfigRequest{ID: cfg.ID, ETag: cfg.ETag, ConfigData: *configData}
					bodyBytes, err := json.Marshal(payload)
					if err != nil {
						log.WithError(err).Error("failed to marshal config for worker")
						continue
					}
					workerReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/config", r.workerURL), bytes.NewReader(bodyBytes))
					if err != nil {
						log.WithError(err).Error("failed to create worker request")
						continue
					}
					workerReq.Header.Set("Content-Type", "application/json")
					// generate correlation id for this forward
					corr := uuid.Must(uuid.NewV7()).String()
					workerReq.Header.Set("X-Correlation-ID", corr)
					if r.apiToken != "" {
						workerReq.Header.Set("Authorization", "Bearer "+r.apiToken)
					}
					wresp, err := client.Do(workerReq)
					if err != nil {
						log.WithError(err).Error("failed to send config to worker")
						continue
					}
					wresp.Body.Close()
					if wresp.StatusCode != http.StatusOK {
						log.Error("worker rejected config", zap.Int("status", wresp.StatusCode))
						continue
					}
					log.Info("configuration forwarded to worker via poll", zap.String("etag", cfg.ETag))
				}
			}
		}
	}()
}

// RegisterHeartbeatPolling starts periodic heartbeat to controller
func (r *Repository) RegisterHeartbeatPolling(ctx context.Context, log *logger.CanonicalLogger, interval time.Duration) {
	if r == nil {
		return
	}
	if interval <= 0 {
		log.Info("heartbeat polling disabled due to non-positive interval")
		return
	}

	ticker := time.NewTicker(interval)

	go func() {
		log.Info("Heartbeat polling started", zap.Duration("interval", interval))
		client := &http.Client{Timeout: 10 * time.Second}
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				log.Info("Heartbeat polling stopped")
				return
			case <-ticker.C:
				// read current stored etag
				r.storeMutex.RLock()
				etag := ""
				agentID := r.agentID
				token := r.apiToken
				if r.store != nil {
					etag = r.store.ETag
				}
				r.storeMutex.RUnlock()

				payload := map[string]string{"config_version": etag, "status": "healthy"}
				body, err := json.Marshal(payload)
				if err != nil {
					log.WithError(err).Error("failed to marshal heartbeat payload")
					continue
				}

				target := fmt.Sprintf("%s/heartbeat", r.controllerURL)
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
				if err != nil {
					log.WithError(err).Error("failed to create heartbeat request")
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				if agentID != "" {
					req.Header.Set("X-Agent-ID", agentID)
				}
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}

				resp, err := client.Do(req)
				if err != nil {
					log.WithError(err).Error("heartbeat request failed")
					continue
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					log.Error("heartbeat not accepted by controller", zap.Int("status", resp.StatusCode), zap.String("agent_id", agentID))
					continue
				}
				log.Info("Heartbeat sent successfully", zap.String("agent_id", agentID), zap.String("config_version", etag))
			}
		}
	}()
}

// SetAgentID sets the agent ID
func (r *Repository) SetAgentID(agentID string) error {
	r.storeMutex.Lock()
	defer r.storeMutex.Unlock()
	if r.store == nil {
		r.store = &StoreData{}
	}
	r.store.AgentID = agentID
	r.agentID = agentID
	return nil
}

// GetAgentID returns the stored agent ID
func (r *Repository) GetAgentID() (string, error) {
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()
	if r.store == nil {
		return "", nil
	}
	return r.store.AgentID, nil
}

// GetCurrentConfig retrieves the current worker configuration
func (r *Repository) GetCurrentConfig() (*models.Configuration, error) {
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()
	if r.store == nil {
		return nil, nil
	}
	return r.store.Config, nil
}

// UpdateConfig updates the worker configuration
func (r *Repository) UpdateConfig(config *models.Configuration) error {
	if r == nil {
		return nil
	}
	r.storeMutex.Lock()
	defer r.storeMutex.Unlock()
	if r.store == nil {
		r.store = &StoreData{}
	}
	r.store.Config = config
	r.store.ETag = config.ETag
	return nil
}

// StartRedisListener starts listening for config update notifications
func (r *Repository) StartRedisListener(ctx context.Context, log *logger.CanonicalLogger) error {
	if r.pubsub == nil {
		log.Info("Redis subscriber not configured, skipping push notifications")
		return nil
	}

	// Start managed connection goroutine
	go r.manageRedisConnection(ctx, log)
	return nil
}

const (
	maxRedisFailures       = 5
	circuitBreakerCooldown = 5 * time.Minute
)

// shouldAttemptRedisReconnect checks if we should try reconnecting to Redis
func (r *Repository) shouldAttemptRedisReconnect() bool {
	r.circuitMutex.Lock()
	defer r.circuitMutex.Unlock()
	if !r.redisCircuitOpen {
		return true
	}
	// If circuit open, allow reconnect attempt after cooldown
	if time.Since(r.lastRedisFailure) > circuitBreakerCooldown {
		r.redisCircuitOpen = false
		r.redisFailures = 0
		return true
	}
	return false
}

// recordRedisFailure increments failure counter and opens circuit if needed
func (r *Repository) recordRedisFailure() {
	r.circuitMutex.Lock()
	defer r.circuitMutex.Unlock()
	r.redisFailures++
	r.lastRedisFailure = time.Now()
	if r.redisFailures >= maxRedisFailures {
		r.redisCircuitOpen = true
	}
}

// recordRedisSuccess resets circuit breaker
func (r *Repository) recordRedisSuccess() {
	r.circuitMutex.Lock()
	defer r.circuitMutex.Unlock()
	r.redisFailures = 0
	r.redisCircuitOpen = false
}

// manageRedisConnection handles Redis connection with circuit breaker and reconnection
func (r *Repository) manageRedisConnection(ctx context.Context, log *logger.CanonicalLogger) {
	channel := "config-updates"
	for {
		if ctx.Err() != nil {
			return
		}

		if !r.shouldAttemptRedisReconnect() {
			// circuit open; wait a bit before checking again
			time.Sleep(10 * time.Second)
			continue
		}

		msgCh, err := r.pubsub.Subscribe(ctx, channel)
		if err != nil {
			log.WithError(err).Error("failed to subscribe to redis channel")
			r.recordRedisFailure()
			// backoff before retrying
			time.Sleep(5 * time.Second)
			continue
		}

		log.Info("Subscribed to Redis config updates channel", zap.String("channel", channel), zap.String("agent_id", r.agentID))
		r.recordRedisSuccess()

		// Listen to messages until subscription breaks
		alive := r.listenToRedis(ctx, log, msgCh)
		if !alive {
			// subscription ended unexpectedly; record failure and attempt reconnect
			r.recordRedisFailure()
			time.Sleep(2 * time.Second)
			continue
		}
	}
}

// listenToRedis listens for Redis messages, returns false if connection is lost
func (r *Repository) listenToRedis(ctx context.Context, log *logger.CanonicalLogger, msgChan <-chan pubsub.Message) bool {
	for {
		select {
		case <-ctx.Done():
			log.Info("Redis listener stopped")
			return true
		case msg, ok := <-msgChan:
			if !ok {
				log.Info("redis message channel closed")
				return false
			}
			var payload struct {
				AgentID       string `json:"agent_id"`
				ETag          string `json:"etag"`
				CorrelationID string `json:"correlation_id"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				log.WithError(err).Error("failed to unmarshal redis message")
				continue
			}
			// If message targets a specific agent and it's not us, skip
			if payload.AgentID != "" && r.agentID != "" && payload.AgentID != r.agentID {
				continue
			}
			if err := r.handleConfigUpdate(ctx, log, payload.ETag, payload.CorrelationID); err != nil {
				log.WithError(err).Error("failed to handle config update notification")
			} else {
				log.Info("received config update notification", zap.String("etag", payload.ETag), zap.String("correlation_id", payload.CorrelationID))
			}
		}
	}
}
