package dpdc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/time/rate"
)

const name = "dpdc"

type Service struct {
	authClient  *datasources.Client
	usageClient *datasources.Client
	config      config.DpdcConfig
	limiter     *rate.Limiter
	apiHits     metric.Int64Counter
	mu          sync.Mutex
	token       string
	tokenExp    time.Time
}

func NewService(cfg config.DpdcConfig) *Service {
	m := otel.Meter("meterbot/dpdc")
	apiHits, _ := m.Int64Counter(
		"dpdc.api.hit",
		metric.WithDescription("Number of successful DPDC balance fetches"),
	)
	return &Service{
		authClient: datasources.NewClient(&datasources.ClientConfig{
			BasePath: cfg.AuthURL,
			Timeout:  cfg.Timeout,
		}),
		usageClient: datasources.NewClient(&datasources.ClientConfig{
			BasePath: cfg.UsageURL,
			Timeout:  cfg.Timeout,
		}),
		config:  cfg,
		limiter: rate.NewLimiter(rate.Limit(cfg.RateLimit), 1),
		apiHits: apiHits,
	}
}

func (s *Service) GetBalance(ctx context.Context, id datasources.Identifier) (datasources.Balance, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return datasources.Balance{}, fmt.Errorf("rate limit wait: %w", err)
	}
	ctx = context.WithValue(ctx, datasources.CtxKeyDatasource, datasources.CtxDatasourceDpdc)

	token, err := s.getToken(ctx)
	if err != nil {
		return datasources.Balance{}, fmt.Errorf("dpdc auth failed: %w", err)
	}

	gql := fmt.Sprintf(`query { postBalanceDetails(input: {customerNumber: "%s", tenantCode: "DPDC"}) { customerName balanceRemaining errorMessage balanceLatestDate} }`, id.AccountNumber)
	in := BalanceQueryRequest{Query: gql}

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("accesstoken", token)
	headers.Set("tenantCode", s.config.TenantCode)

	var gqlResp PostBalanceDetailsResponse
	if err = s.usageClient.Do(ctx, http.MethodPost, "", headers, &in, &gqlResp); err != nil {
		return datasources.Balance{}, err
	}

	output := gqlResp.Data.PostBalanceDetails
	if output.ErrorMessage != "" {
		return datasources.Balance{}, fmt.Errorf("dpdc: %s", output.ErrorMessage)
	}
	slog.DebugContext(ctx, "dpdc balance retrieved", "account_number", id.AccountNumber)
	s.apiHits.Add(ctx, 1, metric.WithAttributes(attribute.String("dpdc.api", "graphql")))

	return datasources.Balance{
		Identifier:  id,
		Balance:     output.BalanceRemaining,
		ReadingTime: datasources.ParseReadingTime(output.BalanceLatestDate),
	}, nil
}

func (s *Service) Name() string { return name }

func (s *Service) getToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && time.Now().Before(s.tokenExp) {
		slog.DebugContext(ctx, "dpdc using cached token")
		return s.token, nil
	}

	if s.config.ClientID == "" || s.config.ClientSecret == "" {
		return "", errors.New("dpdc credentials are blank, check your environment configuration")
	}

	headers := make(http.Header)
	headers.Set("clientId", s.config.ClientID)
	headers.Set("clientSecret", s.config.ClientSecret)
	headers.Set("tenantCode", s.config.TenantCode)

	var auth AuthResponse
	if err := s.authClient.Do(ctx, http.MethodPost, "", headers, map[string]any{}, &auth); err != nil {
		return "", err
	}

	if auth.AccessToken == "" {
		return "", errors.New("dpdc auth: no access_token in response")
	}

	s.token = auth.AccessToken
	s.tokenExp = time.Now().Add(25 * time.Minute)
	slog.DebugContext(ctx, "dpdc new token acquired", "expires_at", s.tokenExp)

	return s.token, nil
}
