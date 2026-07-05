package instance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"
	"time"
)

type proxyState struct {
	mu               sync.Mutex
	target           string
	reverseProxy     *httputil.ReverseProxy
	inflight         atomic.Int64
	lastRequestTime  atomic.Int64
	shuttingDown     atomic.Bool
	healthy          atomic.Bool
}

func newProxyState(host string, port int) *proxyState {
	target := fmt.Sprintf("http://%s:%d", host, port)
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = "http"
			r.URL.Host = fmt.Sprintf("%s:%d", host, port)
			r.Host = r.URL.Host
		},
		FlushInterval: -1,
	}
	p := &proxyState{target: target, reverseProxy: proxy}
	p.lastRequestTime.Store(time.Now().UnixNano())
	return p
}

func (p *proxyState) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.shuttingDown.Load() {
		http.Error(w, "instance is shutting down", http.StatusServiceUnavailable)
		return
	}
	p.inflight.Add(1)
	defer p.inflight.Add(-1)
	p.lastRequestTime.Store(time.Now().UnixNano())
	p.reverseProxy.ServeHTTP(w, r)
}

func (p *proxyState) markHealthy() {
	p.healthy.Store(true)
}

func (p *proxyState) isInflight() bool {
	return p.inflight.Load() > 0
}

func (p *proxyState) getLastRequestTime() time.Time {
	return time.Unix(0, p.lastRequestTime.Load())
}

func (p *proxyState) setShuttingDown() {
	p.shuttingDown.Store(true)
}

func (p *proxyState) WaitForHealthy(ctx context.Context, host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 200 * time.Millisecond

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if healthy, err := pollHealth(host, port); err == nil && healthy {
			return nil
		}

		time.Sleep(interval)
	}
	return fmt.Errorf("instance did not become healthy within %v", timeout)
}

func pollHealth(host string, port int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s:%d/health", host, port), nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
