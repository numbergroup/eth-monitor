package monitor

import (
    "context"
    "strings"
    "testing"
    "time"

    "github.com/numbergroup/eth-monitor/pkg/config"
    "github.com/sirupsen/logrus"
)

type fakePeerRPC struct {
    ret uint64
    err error
}

func (f *fakePeerRPC) PeerCount(ctx context.Context) (uint64, error) { return f.ret, f.err }

func newPeerTestMonitor(t *testing.T, rpc RPCPeerCount, ep config.Endpoint) *PeerCountMonitor {
    t.Helper()
    conf := &config.Config{Log: logrus.New()}
    mon, err := NewPeerCountMonitor(conf, nil, rpc, ep)
    if err != nil {
        t.Fatalf("failed to create monitor: %v", err)
    }
    pm, ok := mon.(*PeerCountMonitor)
    if !ok {
        t.Fatalf("unexpected type: %T", mon)
    }
    return pm
}

func TestPeerCount_OK(t *testing.T) {
    rpc := &fakePeerRPC{ret: 8}
    ep := config.Endpoint{MinPeers: 5, PollDuration: 10 * time.Millisecond}
    m := newPeerTestMonitor(t, rpc, ep)

    if err := m.checkPeerCount(context.Background()); err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if m.lastPeerCount != 8 {
        t.Fatalf("expected lastPeerCount=8, got %d", m.lastPeerCount)
    }
}

func TestPeerCount_BelowThreshold_Error(t *testing.T) {
    rpc := &fakePeerRPC{ret: 2}
    ep := config.Endpoint{MinPeers: 3}
    m := newPeerTestMonitor(t, rpc, ep)

    err := m.checkPeerCount(context.Background())
    if err == nil {
        t.Fatal("expected error when below threshold, got nil")
    }
    if !strings.Contains(err.Error(), "below minimum") {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestPeerCount_RPCError_Propagates(t *testing.T) {
    rpc := &fakePeerRPC{ret: 0, err: assertErr{}}
    ep := config.Endpoint{MinPeers: 1}
    m := newPeerTestMonitor(t, rpc, ep)

    err := m.checkPeerCount(context.Background())
    if err == nil {
        t.Fatal("expected RPC error, got nil")
    }
    if !strings.Contains(err.Error(), "failed to get peer count") {
        t.Fatalf("unexpected error message: %v", err)
    }
}

func TestPeerCount_Name(t *testing.T) {
    rpc := &fakePeerRPC{ret: 5}
    ep := config.Endpoint{Name: "example"}
    m := newPeerTestMonitor(t, rpc, ep)
    if got, want := m.Name(), "PeerCountMonitor::example"; got != want {
        t.Fatalf("unexpected name got %q want %q", got, want)
    }
}

