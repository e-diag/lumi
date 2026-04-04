package worker

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/freeway-vpn/backend/internal/usecase"
)

// NodeHealthWorker периодически проверяет ноды (TCP + TLS handshake) и обновляет метрики для сортировки в подписке.
type NodeHealthWorker struct {
	nodeUC   usecase.NodeUseCase
	interval time.Duration
	timeout  time.Duration
}

func NewNodeHealthWorker(nodeUC usecase.NodeUseCase) *NodeHealthWorker {
	return &NodeHealthWorker{
		nodeUC:   nodeUC,
		interval: 5 * time.Minute,
		timeout:  5 * time.Second,
	}
}

func (w *NodeHealthWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.run(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *NodeHealthWorker) run(ctx context.Context) {
	nodes, err := w.nodeUC.GetAllNodes(ctx)
	if err != nil {
		slog.Error("node health worker: get nodes failed", "error", err)
		return
	}
	slog.Info("node health worker: probe cycle", "nodes", len(nodes))

	for _, n := range nodes {
		if n == nil || n.Host == "" {
			continue
		}
		addr := fmt.Sprintf("%s:%d", n.Host, n.Port)
		sni := n.SNI
		if sni == "" {
			sni = n.Host
		}

		start := time.Now()
		err := w.probeNode(ctx, addr, sni)
		n.ProbeTotal++

		if err != nil {
			n.FailCount++
			n.HealthScore -= 12
			if n.HealthScore < 0 {
				n.HealthScore = 0
			}
			if n.FailCount >= 2 && n.Active {
				n.Active = false
				slog.Error("node down", "node", n.Name, "host", n.Host, "port", n.Port, "error", err)
			}
			_ = w.nodeUC.UpdateNode(ctx, n)
			select {
			case <-ctx.Done():
				return
			case <-time.After(15 * time.Millisecond):
			}
			continue
		}

		latency := int(time.Since(start).Milliseconds())
		if n.LatencyMs > 0 {
			n.LatencyMs = int(0.65*float64(n.LatencyMs) + 0.35*float64(latency))
		} else {
			n.LatencyMs = latency
		}
		n.FailCount = 0
		n.ProbeOK++
		n.HealthScore += 3
		if n.HealthScore > 100 {
			n.HealthScore = 100
		}
		if !n.Active {
			slog.Info("node recovered", "node", n.Name, "host", n.Host, "port", n.Port, "latency_ms", n.LatencyMs)
		}
		n.Active = true
		_ = w.nodeUC.UpdateNode(ctx, n)

		select {
		case <-ctx.Done():
			return
		case <-time.After(15 * time.Millisecond):
		}
	}
}

func (w *NodeHealthWorker) probeNode(ctx context.Context, addr, serverName string) error {
	dialer := &net.Dialer{Timeout: w.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp: %w", err)
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	})
	_ = tlsConn.SetDeadline(time.Now().Add(w.timeout))
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return fmt.Errorf("tls: %w", err)
	}
	_ = tlsConn.Close()
	return nil
}
