package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type metrics struct {
	starts       atomic.Uint64
	exits        atomic.Uint64
	running      atomic.Int64
	lastExit     atomic.Int64
	startTime    time.Time
	hardware     string
	runtime      string
	accelerator  string
	variant      string
	profile      string
	capabilities string
	o11y         string
}

type dogStatsD struct {
	conn net.Conn
	tags string
}

func startMetricsServer(addr string, stats *metrics) (*http.Server, net.Listener, error) {
	if addr == "" {
		return nil, nil, nil
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if stats.running.Load() == 1 {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = io.WriteString(w, stats.prometheus())
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logEvent("error", "metrics_server_stopped", map[string]string{"error": err.Error()})
		}
	}()
	return server, listener, nil
}

func shutdownServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func (m *metrics) prometheus() string {
	labels := "hardware=\"" + prometheusLabel(m.hardware) + "\",runtime=\"" + prometheusLabel(m.runtime) + "\",accelerator=\"" + prometheusLabel(m.accelerator) + "\",variant=\"" + prometheusLabel(m.variant) + "\",profile=\"" + prometheusLabel(m.profile) + "\",capabilities=\"" + prometheusLabel(m.capabilities) + "\",o11y=\"" + prometheusLabel(m.o11y) + "\""
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP kait_info Kait runtime build and configuration information.\n# TYPE kait_info gauge\nkait_info{version=\"%s\",%s} 1\n", version, labels)
	fmt.Fprintf(&b, "# HELP kait_agent_starts_total Number of child process starts.\n# TYPE kait_agent_starts_total counter\nkait_agent_starts_total{%s} %d\n", labels, m.starts.Load())
	fmt.Fprintf(&b, "# HELP kait_agent_exits_total Number of child process exits.\n# TYPE kait_agent_exits_total counter\nkait_agent_exits_total{%s} %d\n", labels, m.exits.Load())
	fmt.Fprintf(&b, "# HELP kait_agent_running Whether the child process is running.\n# TYPE kait_agent_running gauge\nkait_agent_running{%s} %d\n", labels, m.running.Load())
	fmt.Fprintf(&b, "# HELP kait_agent_last_exit_code Last child process exit code.\n# TYPE kait_agent_last_exit_code gauge\nkait_agent_last_exit_code{%s} %d\n", labels, m.lastExit.Load())
	fmt.Fprintf(&b, "# HELP process_start_time_seconds Unix time when the Kait runtime started.\n# TYPE process_start_time_seconds gauge\nprocess_start_time_seconds %.3f\n", float64(m.startTime.UnixNano())/1e9)
	return b.String()
}

func newDogStatsD(cfg config) (*dogStatsD, error) {
	if cfg.o11y != "datadog" {
		return nil, nil
	}
	host := envOr("KAIT_DD_AGENT_HOST", envOr("DD_AGENT_HOST", "127.0.0.1"))
	port := envOr("KAIT_DD_DOGSTATSD_PORT", envOr("DD_DOGSTATSD_PORT", "8125"))
	conn, err := net.Dial("udp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	tags := "kait_hardware:" + cfg.hardware + ",kait_runtime:" + cfg.identity.Runtime + ",kait_accelerator:" + cfg.identity.Accelerator + ",kait_variant:" + cfg.variant + ",kait_o11y:datadog"
	return &dogStatsD{conn: conn, tags: tags}, nil
}

func (d *dogStatsD) Close() {
	_ = d.conn.Close()
}

func (d *dogStatsD) gauge(name string, value int) {
	d.send(name, strconv.Itoa(value)+"|g")
}

func (d *dogStatsD) count(name string, value int) {
	d.send(name, strconv.Itoa(value)+"|c")
}

func (d *dogStatsD) send(name, value string) {
	_, _ = fmt.Fprintf(d.conn, "%s:%s|#%s\n", name, value, d.tags)
}

func prometheusLabel(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return strings.ReplaceAll(value, "\n", "\\n")
}
