package worker

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/freeway-vpn/backend/internal/repository"
)

// DomainHealthWorker проверяет домены из node_domains (TCP+TLS) и помечает заблокированные.
type DomainHealthWorker struct {
	nodeRepo repository.NodeRepository
	interval time.Duration
}

// NewDomainHealthWorker создаёт воркер проверки доменов.
func NewDomainHealthWorker(nodeRepo repository.NodeRepository) *DomainHealthWorker {
	return &DomainHealthWorker{
		nodeRepo: nodeRepo,
		interval: 3 * time.Minute,
	}
}

func (w *DomainHealthWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.run(ctx)
	for {
		select {
		case <-ticker.C:
			w.run(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *DomainHealthWorker) run(ctx context.Context) {
	rows, err := w.nodeRepo.ListActiveNodeDomains(ctx)
	if err != nil {
		slog.Error("domain health: list domains", "error", err)
		return
	}
	for _, d := range rows {
		if d == nil || d.Domain == "" {
			continue
		}
		ok, err := probeTLSHandshake(ctx, d.Domain, 5*time.Second)
		now := time.Now()
		d.LastCheckedAt = &now
		if ok {
			d.ConsecutiveFails = 0
			d.IsBlocked = false
			d.LastSuccessAt = &now
		} else {
			d.ConsecutiveFails++
			if d.ConsecutiveFails >= 3 {
				d.IsBlocked = true
				slog.Warn("domain marked blocked", "domain", d.Domain, "fails", d.ConsecutiveFails, "error", err)
			}
		}
		if err := w.nodeRepo.UpdateNodeDomain(ctx, d); err != nil {
			slog.Error("domain health: save", "domain", d.Domain, "error", err)
		}
	}
}

func probeTLSHandshake(ctx context.Context, host string, timeout time.Duration) (bool, error) {
	d := net.JoinHostPort(host, "443")
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", d)
	if err != nil {
		return false, fmt.Errorf("tcp: %w", err)
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	})
	defer tlsConn.Close()
	_ = tlsConn.SetDeadline(time.Now().Add(timeout))
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return false, fmt.Errorf("tls: %w", err)
	}
	return true, nil
}
