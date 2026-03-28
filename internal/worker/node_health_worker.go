package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/freeway-vpn/backend/internal/usecase"
)

type NodeHealthWorker struct {
	nodeUC   usecase.NodeUseCase
	interval time.Duration
	timeout  time.Duration
}

func NewNodeHealthWorker(nodeUC usecase.NodeUseCase) *NodeHealthWorker {
	return &NodeHealthWorker{
		nodeUC:   nodeUC,
		interval: 5 * time.Minute,
		timeout:  3 * time.Second,
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

	for _, n := range nodes {
		if n == nil {
			continue
		}
		addr := fmt.Sprintf("%s:%d", n.Host, n.Port)
		start := time.Now()
		c, err := (&net.Dialer{Timeout: w.timeout}).DialContext(ctx, "tcp", addr)
		if err != nil {
			n.FailCount++
			if n.FailCount >= 2 && n.Active {
				n.Active = false
				slog.Error("node down", "node", n.Name, "host", n.Host, "port", n.Port)
			}
			_ = w.nodeUC.UpdateNode(ctx, n)
			continue
		}
		_ = c.Close()

		latency := time.Since(start)
		n.LatencyMs = int(latency.Milliseconds())
		if !n.Active {
			slog.Info("node recovered", "node", n.Name, "host", n.Host, "port", n.Port, "latency_ms", n.LatencyMs)
		}
		n.Active = true
		n.FailCount = 0
		_ = w.nodeUC.UpdateNode(ctx, n)
	}
}

