package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/repository"
)

type mockControllerClient struct {
	regResp     *models.RegistrationResponse
	cfgResp     *models.Configuration
	etag        string
	notModified bool
}

func (m *mockControllerClient) Register(ctx context.Context, hostname, version, startTime string) (*models.RegistrationResponse, error) {
	if m.regResp == nil {
		return nil, errors.New("register failed")
	}
	return m.regResp, nil
}
func (m *mockControllerClient) GetConfiguration(ctx context.Context, agentID, pollURL, ifNoneMatch string) (*models.Configuration, string, bool, error) {
	if m.notModified {
		return nil, "", true, nil
	}
	return m.cfgResp, m.etag, false, nil
}

type mockWorkerClient struct {
	sent []*models.Configuration
}

func (m *mockWorkerClient) SendConfiguration(ctx context.Context, config *models.Configuration) error {
	m.sent = append(m.sent, config)
	return nil
}

func TestRegisterWithControllerStoresAgentID(t *testing.T) {
	repo := repository.NewRepository()
	ctrl := &mockControllerClient{regResp: &models.RegistrationResponse{AgentID: "agent-123"}}
	worker := &mockWorkerClient{}
	cfg := &config.AgentConfig{RegistrationMaxRetries: 1, RegistrationInitialBackoff: 1}

	uc := NewUseCase(ctrl, repo, worker, cfg)
	if _, err := uc.RegisterWithController(context.Background(), "host-1", "start"); err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}
	id, _ := repo.GetAgentID()
	if id != "agent-123" {
		t.Fatalf("expected agent id stored, got %s", id)
	}
}

func TestFetchConfigurationUpdatesRepoAndNotifiesWorker(t *testing.T) {
	repo := repository.NewRepository()
	ctrl := &mockControllerClient{cfgResp: &models.Configuration{ID: 1, ConfigData: "data"}, etag: "v2", notModified: false}
	worker := &mockWorkerClient{}
	cfg := &config.AgentConfig{ControllerURL: "http://x"}

	uc := NewUseCase(ctrl, repo, worker, cfg)
	cfgRes, notModified, err := uc.FetchConfiguration(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notModified {
		t.Fatalf("expected notModified=false")
	}
	if cfgRes == nil || cfgRes.ConfigData != "data" {
		t.Fatalf("expected configuration returned and stored")
	}
	if len(worker.sent) != 1 {
		t.Fatalf("expected worker to be notified once, got %d", len(worker.sent))
	}
}
