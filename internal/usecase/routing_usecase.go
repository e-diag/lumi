package usecase

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
)

var (
	antifilterDomainsURL = "https://antifilter.download/list/domains.lst"
	antifilterCIDRURL    = "https://antifilter.download/list/allyouneed.lst"
)

type routingUseCase struct {
	repo       repository.RoutingRepository
	httpClient *http.Client
}

func NewRoutingUseCase(repo repository.RoutingRepository) RoutingUseCase {
	return &routingUseCase{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (uc *routingUseCase) GetLists(ctx context.Context) (*RoutingLists, error) {
	list, err := uc.repo.GetRoutingList(ctx)
	if err != nil {
		return nil, fmt.Errorf("routing usecase: get lists: %w", err)
	}

	return &RoutingLists{
		Version:          time.Now().UTC().Format("2006-01-02"),
		ProxyEU:          list.ProxyEU,
		ProxyUSA:         list.ProxyUSA,
		Direct:           list.Direct,
		Manual:           list.Manual,
		DirectStrictMode: true,
		Meta: RoutingMeta{
			CDNFallbackExcludesDirect: true,
			DirectDomainsNote: "Российские домены идут напрямую даже при включенном CDN-fallback. " +
				"Не маршрутизируй их через CDN во избежание бана белых IP-подсетей.",
		},
	}, nil
}

func (uc *routingUseCase) UpdateFromAntifilter(ctx context.Context) error {
	euDomains, err := uc.fetchList(ctx, antifilterDomainsURL)
	if err != nil {
		return fmt.Errorf("routing usecase: fetch domains: %w", err)
	}
	euCIDR, err := uc.fetchList(ctx, antifilterCIDRURL)
	if err != nil {
		return fmt.Errorf("routing usecase: fetch cidr: %w", err)
	}

	if err := uc.repo.SaveDomains(ctx, "antifilter", domain.ActionProxyEU, append(euDomains, euCIDR...)); err != nil {
		return fmt.Errorf("routing usecase: save antifilter: %w", err)
	}
	if err := uc.repo.SaveDomains(ctx, "builtin_ai", domain.ActionProxyUSA, domain.AIServiceDomains); err != nil {
		return fmt.Errorf("routing usecase: save ai domains: %w", err)
	}
	if err := uc.repo.SaveDomains(ctx, "builtin_direct", domain.ActionDirect, domain.DirectDomains); err != nil {
		return fmt.Errorf("routing usecase: save direct domains: %w", err)
	}

	slog.Info("routing lists updated", "proxy_eu_count", len(euDomains), "source", "antifilter.download")
	return nil
}

func (uc *routingUseCase) AddDomain(ctx context.Context, domainName string, action domain.RouteAction) error {
	if err := uc.repo.AddManualDomain(ctx, domainName, action); err != nil {
		return fmt.Errorf("routing usecase: add domain: %w", err)
	}
	return nil
}

func (uc *routingUseCase) RemoveDomain(ctx context.Context, domainName string) error {
	if err := uc.repo.DeleteManualDomain(ctx, domainName); err != nil {
		return fmt.Errorf("routing usecase: remove domain: %w", err)
	}
	return nil
}

func (uc *routingUseCase) fetchList(ctx context.Context, endpoint string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := make([]string, 0, 1024)
	seen := map[string]struct{}{}
	for sc.Scan() {
		line := strings.TrimSpace(strings.ToLower(sc.Text()))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "http://")
		line = strings.TrimPrefix(line, "https://")
		line = strings.TrimPrefix(line, "www.")
		if i := strings.IndexByte(line, '/'); i >= 0 {
			line = line[:i]
		}
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan list: %w", err)
	}
	return out, nil
}

