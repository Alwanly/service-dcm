package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"

	"github.com/Alwanly/service-distribute-management/internal/models"
	dto "github.com/Alwanly/service-distribute-management/internal/server/worker/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/worker/repository"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"
	"go.uber.org/zap"
)

type UseCaseInterface interface {
	ReceiveConfig(ctx context.Context, req *dto.ReceiveConfigRequest) wrapper.JSONResult
	HitRequest(ctx context.Context) wrapper.JSONResult
	GetCurrentConfig() *models.ConfigData
}

type UseCase struct {
	repo       repository.IRepository
	httpClient *http.Client
}

func NewUseCase(repo repository.IRepository, timeout time.Duration) UseCaseInterface {
	return &UseCase{
		repo: repo,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (uc *UseCase) ReceiveConfig(ctx context.Context, req *dto.ReceiveConfigRequest) wrapper.JSONResult {

	configData, err := json.Marshal(req.ConfigData)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err))
		return wrapper.ResponseSuccess(http.StatusConflict, "Failed validate configData")
	}

	// Create worker configuration model
	config := &models.Configuration{
		ID:         req.ID,
		ETag:       req.ETag,
		ConfigData: string(configData),
	}

	// Update configuration in repository
	if err := uc.repo.UpdateConfig(config); err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.JSONResult{
			Code:    fiber.StatusInternalServerError,
			Success: false,
			Message: "Failed to update configuration",
			Data:    nil,
		}
	}

	logger.AddToContext(ctx,
		zap.Bool(logger.FieldSuccess, true),
		zap.String(logger.FieldETag, req.ETag),
	)

	return wrapper.ResponseSuccess(http.StatusOK, nil)
}

func (uc *UseCase) HitRequest(ctx context.Context) wrapper.JSONResult {
	// Get current configuration
	data, err := uc.repo.GetCurrentConfig()
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to get configuration", nil)
	}

	if data == nil {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, false), zap.String(logger.FieldProxyStatus, "no_config"))
		return wrapper.ResponseFailed(http.StatusBadRequest, "no configuration available", nil)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, data.Config.URL, nil)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to create request", nil)
	}
	client := uc.httpClient
	if data.Config.Proxy != "" {
		proxyURL, err := parseProxyURL(data.Config.Proxy)
		if err != nil {
			logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
			return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to parse proxy", nil)
		}

		transport := &http.Transport{
			Proxy:                 http.ProxyURL(proxyURL),
			DisableKeepAlives:     true,
			DisableCompression:    false,
			MaxIdleConns:          0,
			MaxIdleConnsPerHost:   -1,
			IdleConnTimeout:       0,
			TLSHandshakeTimeout:   30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		client = &http.Client{
			Timeout:   uc.httpClient.Timeout,
			Transport: transport,
		}

		logger.AddToContext(ctx,
			zap.String("proxy_url", proxyURL.Host),
			zap.Bool("proxy_configured", true),
		)
	}

	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Connection", "close")
	// Perform HTTP request
	resp, err := client.Do(req)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to perform request", nil)
	}
	defer resp.Body.Close()
	logger.AddToContext(ctx,
		zap.Bool(logger.FieldSuccess, true),
		zap.String(logger.FieldTargetURL, data.Config.URL),
		zap.Int("status_code", resp.StatusCode),
	)

	var respBody []byte
	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to read response body", nil)
	}

	// Parse HTML to extract IP address from class "ip-address"
	respData, err := extractIPFromHTML(respBody)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to parse HTML response", nil)
	}

	response := &dto.HitResponse{
		ETag: data.ETag,
		URL:  data.Config.URL,
		Data: respData,
	}
	return wrapper.ResponseSuccess(http.StatusOK, response)
}

func (uc *UseCase) GetCurrentConfig() *models.ConfigData {
	data, err := uc.repo.GetCurrentConfig()
	if err != nil || data == nil {
		return nil
	}
	return &data.Config
}

func extractIPFromHTML(htmlData []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlData))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	ipValue, exists := doc.Find("input[name='ip']").First().Attr("value")
	if !exists || ipValue == "" {
		return "", fmt.Errorf("input element with name='ip' or its value not found in HTML")
	}

	return strings.TrimSpace(ipValue), nil
}

func parseProxyURL(proxy string) (*url.URL, error) {
	// Handle format: host:port:username:password
	parts := strings.Split(proxy, ":")
	if len(parts) == 4 {
		host := parts[0]
		port := parts[1]
		username := parts[2]
		password := parts[3]

		// Construct proxy URL with authentication: http://username:password@host:port
		proxyURLString := fmt.Sprintf("http://%s:%s@%s:%s", username, password, host, port)
		return url.Parse(proxyURLString)
	}

	// Handle standard format: http://host:port or host:port
	if !strings.HasPrefix(proxy, "http://") && !strings.HasPrefix(proxy, "https://") {
		proxy = "http://" + proxy
	}

	return url.Parse(proxy)
}
