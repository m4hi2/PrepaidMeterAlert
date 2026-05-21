package desco

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/time/rate"
)

const name = "desco"

const (
	unifiedBalancePath = "/api/unified/customer/getBalance"
	tkdesBalancePath   = "/api/tkdes/customer/getBalance"
)

const (
	paramAccountNo = "accountNo"
	paramMeterNo   = "meterNo"
)

type apiResult struct {
	bal    datasources.Balance
	source string
	err    error
}

type Service struct {
	client  *datasources.Client
	limiter *rate.Limiter
	apiHits metric.Int64Counter
}

func NewService(cfg config.DescoConfig) *Service {
	m := otel.Meter("meterbot/desco")
	apiHits, _ := m.Int64Counter(
		"desco.api.hit",
		metric.WithDescription("Number of successful balance fetches per DESCO API endpoint"),
	)
	return &Service{
		client: datasources.NewClient(&datasources.ClientConfig{
			BasePath:   cfg.BasePath,
			Timeout:    cfg.Timeout,
			Retry:      cfg.Retry,
			RetryDelay: cfg.RetryDelay,
		}),
		limiter: rate.NewLimiter(rate.Limit(cfg.RateLimit), 1),
		apiHits: apiHits,
	}
}

func (s *Service) GetBalance(ctx context.Context, id datasources.Identifier) (datasources.Balance, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return datasources.Balance{}, fmt.Errorf("rate limit wait: %w", err)
	}
	ctx = context.WithValue(ctx, datasources.CtxKeyDatasource, datasources.CtxDatasourceDesco)

	r := s.fetchFirstBalance(ctx, id)
	if r.err != nil {
		return datasources.Balance{}, r.err
	}
	s.apiHits.Add(ctx, 1, metric.WithAttributes(attribute.String("desco.api", r.source)))
	return r.bal, nil
}

func (s *Service) Name() string {
	return name
}

// fetchFirstBalance hits two desco endpoints since desco partitions users on two different
// endpoints. Whichever endpoint gives correct data is returned.
func (s *Service) fetchFirstBalance(ctx context.Context, id datasources.Identifier) apiResult {
	fanCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan apiResult, 2)
	go func() { results <- s.callAPI(fanCtx, id, unifiedBalancePath, "unified") }()
	go func() { results <- s.callAPI(fanCtx, id, tkdesBalancePath, "tkdes") }()

	var errs []error
	for range 2 {
		if r := <-results; r.err == nil {
			return r
		} else {
			errs = append(errs, r.err)
		}
	}
	return apiResult{err: errors.Join(errs...)}
}

func (s *Service) callAPI(ctx context.Context, id datasources.Identifier, path, source string) apiResult {
	var resp GetBalanceResp
	err := s.client.Do(ctx, http.MethodGet, path+"?"+buildQuery(id), nil, nil, &resp)
	if err == nil && resp.Code != http.StatusOK {
		err = fmt.Errorf("upstream code %d: %s", resp.Code, resp.Desc)
	}
	if err != nil {
		return apiResult{err: fmt.Errorf("%s: %w", source, err), source: source}
	}
	outID := id
	outID.AccountNumber, outID.MeterNumber = resp.Data.AccountNo, resp.Data.MeterNo

	var readingTime *time.Time
	if t, parseErr := parseReadingTime(resp.Data.ReadingTime); parseErr == nil {
		readingTime = &t
	} else {
		slog.DebugContext(ctx, "desco: could not parse reading time", "raw", resp.Data.ReadingTime, "error", parseErr)
	}

	return apiResult{
		bal:    datasources.Balance{Identifier: outID, Balance: resp.Data.Balance, ReadingTime: readingTime},
		source: source,
	}
}

func buildQuery(id datasources.Identifier) string {
	q := url.Values{}
	q.Set(paramAccountNo, id.AccountNumber)
	q.Set(paramMeterNo, id.MeterNumber)
	return q.Encode()
}

var readingTimeFormats = []string{
	"2006-01-02",
	time.DateTime,
}

func parseReadingTime(raw string) (time.Time, error) {
	for _, f := range readingTimeFormats {
		if t, err := time.Parse(f, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported reading time format: %q", raw)
}
