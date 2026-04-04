package usecase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
)

type fakeRoutingRepo struct {
	list *domain.RoutingList
}

func (f *fakeRoutingRepo) GetAll(context.Context) ([]*domain.RoutingRule, error)    { return nil, nil }
func (f *fakeRoutingRepo) GetActive(context.Context) ([]*domain.RoutingRule, error) { return nil, nil }
func (f *fakeRoutingRepo) Create(context.Context, *domain.RoutingRule) error        { return nil }
func (f *fakeRoutingRepo) Update(context.Context, *domain.RoutingRule) error        { return nil }
func (f *fakeRoutingRepo) Delete(context.Context, uuid.UUID) error                  { return nil }
func (f *fakeRoutingRepo) GetVersion(context.Context) (string, error)               { return "2026-03-27", nil }
func (f *fakeRoutingRepo) AddManualDomain(context.Context, string, domain.RouteAction) error {
	return nil
}
func (f *fakeRoutingRepo) DeleteManualDomain(context.Context, string) error { return nil }
func (f *fakeRoutingRepo) SaveDomains(context.Context, string, domain.RouteAction, []string) error {
	return nil
}
func (f *fakeRoutingRepo) GetRoutingList(context.Context) (*domain.RoutingList, error) {
	return f.list, nil
}

func TestRoutingUseCase_GetLists(t *testing.T) {
	repo := &fakeRoutingRepo{
		list: &domain.RoutingList{
			Version:  "2026-03-27",
			ProxyEU:  []string{"instagram.com"},
			ProxyUSA: []string{"openai.com"},
			Direct:   []string{"yandex.ru"},
			Manual:   []string{"custom.example"},
		},
	}
	uc := NewRoutingUseCase(repo)
	got, err := uc.GetLists(context.Background())
	if err != nil {
		t.Fatalf("GetLists error: %v", err)
	}
	if len(got.ProxyUSA) == 0 || got.ProxyUSA[0] != "openai.com" {
		t.Fatalf("expected AI domain in proxy_usa")
	}
	if len(got.Direct) == 0 || got.Direct[0] != "yandex.ru" {
		t.Fatalf("expected direct domain")
	}
}

func TestRoutingUseCase_UpdateFromAntifilter_HTTPError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/domains.lst", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "oops", http.StatusInternalServerError)
	})
	mux.HandleFunc("/allyouneed.lst", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("1.1.1.0/24\n"))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	oldDomainsURL := antifilterDomainsURL
	oldCIDRURL := antifilterCIDRURL
	antifilterDomainsURL = ts.URL + "/domains.lst"
	antifilterCIDRURL = ts.URL + "/allyouneed.lst"
	defer func() {
		antifilterDomainsURL = oldDomainsURL
		antifilterCIDRURL = oldCIDRURL
	}()

	repo := &capturingRoutingRepo{}
	uc := NewRoutingUseCase(repo)
	err := uc.UpdateFromAntifilter(context.Background())
	if err == nil {
		t.Fatal("expected error on HTTP failure")
	}
	if len(repo.calls) != 0 {
		t.Fatalf("expected no save calls on HTTP error, got %d", len(repo.calls))
	}
}

type saveCall struct {
	source string
	action domain.RouteAction
	count  int
}

type capturingRoutingRepo struct {
	calls []saveCall
}

func (f *capturingRoutingRepo) GetAll(context.Context) ([]*domain.RoutingRule, error) {
	return nil, nil
}
func (f *capturingRoutingRepo) GetActive(context.Context) ([]*domain.RoutingRule, error) {
	return nil, nil
}
func (f *capturingRoutingRepo) Create(context.Context, *domain.RoutingRule) error { return nil }
func (f *capturingRoutingRepo) Update(context.Context, *domain.RoutingRule) error { return nil }
func (f *capturingRoutingRepo) Delete(context.Context, uuid.UUID) error           { return nil }
func (f *capturingRoutingRepo) GetVersion(context.Context) (string, error)        { return "", nil }
func (f *capturingRoutingRepo) AddManualDomain(context.Context, string, domain.RouteAction) error {
	return nil
}
func (f *capturingRoutingRepo) DeleteManualDomain(context.Context, string) error { return nil }
func (f *capturingRoutingRepo) GetRoutingList(context.Context) (*domain.RoutingList, error) {
	return &domain.RoutingList{}, nil
}
func (f *capturingRoutingRepo) SaveDomains(_ context.Context, source string, action domain.RouteAction, domains []string) error {
	f.calls = append(f.calls, saveCall{source: source, action: action, count: len(domains)})
	return nil
}

func TestRoutingUseCase_UpdateFromAntifilter(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/domains.lst", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#comment\ninstagram.com\ninstagram.com\nfacebook.com\n"))
	})
	mux.HandleFunc("/allyouneed.lst", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("1.1.1.0/24\n\n#x\n"))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	oldDomainsURL := antifilterDomainsURL
	oldCIDRURL := antifilterCIDRURL
	antifilterDomainsURL = ts.URL + "/domains.lst"
	antifilterCIDRURL = ts.URL + "/allyouneed.lst"
	defer func() {
		antifilterDomainsURL = oldDomainsURL
		antifilterCIDRURL = oldCIDRURL
	}()

	repo := &capturingRoutingRepo{}
	uc := NewRoutingUseCase(repo).(*routingUseCase)
	uc.httpClient.Timeout = 5 * time.Second

	if err := uc.UpdateFromAntifilter(context.Background()); err != nil {
		t.Fatalf("UpdateFromAntifilter error: %v", err)
	}
	if len(repo.calls) < 3 {
		t.Fatalf("expected 3 save calls, got %d", len(repo.calls))
	}
}
