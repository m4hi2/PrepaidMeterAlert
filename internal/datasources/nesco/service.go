package nesco

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/net/html"
	"golang.org/x/time/rate"
)

const name = "nesco"

type Service struct {
	client  *datasources.Client
	limiter *rate.Limiter
	apiHits metric.Int64Counter
}

func NewService(cfg config.NescoConfig) *Service {
	jar, _ := cookiejar.New(nil)

	m := otel.Meter("meterbot/nesco")
	apiHits, _ := m.Int64Counter(
		"nesco.api.hit",
		metric.WithDescription("Number of successful NESCO balance fetches"),
	)

	return &Service{
		client: datasources.NewClient(&datasources.ClientConfig{
			BasePath:   cfg.BasePath,
			Timeout:    cfg.Timeout,
			Retry:      cfg.Retry,
			RetryDelay: cfg.RetryDelay,
			Jar:        jar,
		}),
		limiter: rate.NewLimiter(rate.Limit(cfg.RateLimit), 1),
		apiHits: apiHits,
	}
}

func (s *Service) GetBalance(ctx context.Context, id datasources.Identifier) (datasources.Balance, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return datasources.Balance{}, fmt.Errorf("rate limit wait: %w", err)
	}

	ctx = context.WithValue(ctx, datasources.CtxKeyDatasource, datasources.CtxDatasourceNesco)

	if err := s.switchToEnglish(ctx); err != nil {
		return datasources.Balance{}, fmt.Errorf("switch language: %w", err)
	}

	token, err := s.getCSRFToken(ctx)
	if err != nil {
		return datasources.Balance{}, fmt.Errorf("get csrf token: %w", err)
	}

	resp, err := s.fetchBalance(ctx, id.AccountNumber, token)
	if err != nil {
		return datasources.Balance{}, err
	}

	s.apiHits.Add(ctx, 1, metric.WithAttributes(attribute.String("nesco.api", "panel")))

	outID := id
	if resp.Data.MeterNo != "" {
		outID.MeterNumber = resp.Data.MeterNo
	}

	var balance float64
	if resp.Data.Balance != "" {
		balance, err = strconv.ParseFloat(strings.TrimSpace(resp.Data.Balance), 64)
		if err != nil {
			slog.ErrorContext(ctx, "invalid balance", "balance", resp.Data.Balance, "error", err)
			balance = 0
		}
	}

	return datasources.Balance{
		Identifier: outID,
		Balance:    balance,
	}, nil
}

func (s *Service) Name() string {
	return name
}

func (s *Service) switchToEnglish(ctx context.Context) error {
	headers := make(http.Header)
	headers.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.DoRaw(ctx, http.MethodGet, languageEn, headers, nil)
	if err != nil {
		return fmt.Errorf("language switch request: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (s *Service) getCSRFToken(ctx context.Context) (string, error) {
	resp, err := s.client.DoRaw(ctx, http.MethodGet, panelPath, nil, nil)
	if err != nil {
		return "", fmt.Errorf("get csrf token request: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}

	var token string
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, content string
			for _, attr := range n.Attr {
				if attr.Key == "name" {
					name = attr.Val
				}
				if attr.Key == "content" {
					content = attr.Val
				}
			}
			if name == "csrf-token" && content != "" {
				token = content
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)

	if token == "" {
		return "", fmt.Errorf("csrf-token meta tag not found")
	}
	slog.DebugContext(ctx, "nesco csrf token extracted", "length", len(token))
	return token, nil
}

func (s *Service) fetchBalance(ctx context.Context, custNo, token string) (*NescoBalanceResp, error) {
	form := url.Values{}
	form.Set(paramToken, token)
	form.Set(paramCustNo, custNo)
	form.Set(paramSubmit, submitRecharge)

	headers := make(http.Header)
	headers.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.DoRaw(ctx, http.MethodPost, panelPath, headers, []byte(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected upstream status %d", resp.StatusCode)
	}

	return parseBalancePage(ctx, resp.Body)
}

// parseBalancePage extracts balance and other fields from HTML (same logic as python-nesco)
func parseBalancePage(ctx context.Context, body io.Reader) (*NescoBalanceResp, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse response html: %w", err)
	}

	data := make(map[string]string)
	var currentLabel string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "label" {
				currentLabel = strings.TrimSpace(n.FirstChild.Data)
				currentLabel = strings.ReplaceAll(currentLabel, "\n", " ")
			}
			if n.Data == "input" {
				for _, attr := range n.Attr {
					if attr.Key == "value" && currentLabel != "" {
						data[currentLabel] = strings.TrimSpace(attr.Val)
						currentLabel = ""
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if len(data) == 0 {
		return nil, fmt.Errorf("no data extracted from NESCO response")
	}

	acc, ok1 := data[AccountNumber]
	meter, ok2 := data[MeterNumber]
	balanceStr, ok3 := data[Balance]
	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf(
			"missing required NESCO fields: account=%v meter=%v balance=%v",
			ok1,
			ok2,
			ok3,
		)
	}

	slog.DebugContext(context.Background(), "nesco balance extracted", "data", data)

	return &NescoBalanceResp{
		Code: http.StatusOK,
		Data: struct {
			AccountNo string `json:"accountNo"`
			MeterNo   string `json:"meterNo"`
			Balance   string `json:"balance"`
		}{
			AccountNo: acc,
			MeterNo:   meter,
			Balance:   balanceStr,
		},
	}, nil
}
