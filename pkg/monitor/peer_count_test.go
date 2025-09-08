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
    // Since 8 > 5+2, the monitor should consider we've been above min at least once
    if !m.hasEverBeenAboveMin {
        t.Fatalf("expected hasEverBeenAboveMin=true, got false")
    }
}

func TestPeerCount_BelowThreshold_NoErrorBeforeEverAbove(t *testing.T) {
    rpc := &fakePeerRPC{ret: 2}
    ep := config.Endpoint{MinPeers: 3}
    m := newPeerTestMonitor(t, rpc, ep)

    err := m.checkPeerCount(context.Background())
    if err != nil {
        t.Fatalf("expected no error before ever being above min, got %v", err)
    }
    if m.hasEverBeenAboveMin {
        t.Fatalf("expected hasEverBeenAboveMin=false at startup, got true")
    }
}

func TestPeerCount_AlertsAfterEverAboveAndThenBelow(t *testing.T) {
    rpc := &fakePeerRPC{ret: 7}
    ep := config.Endpoint{MinPeers: 3}
    m := newPeerTestMonitor(t, rpc, ep)

    // First, go above MinPeers+2 to set the flag
    if err := m.checkPeerCount(context.Background()); err != nil {
        t.Fatalf("unexpected error on initial high peer count: %v", err)
    }
    if !m.hasEverBeenAboveMin {
        t.Fatalf("expected hasEverBeenAboveMin=true after high peer count")
    }

    // Now drop below MinPeers and expect an alert
    rpc.ret = 2
    err := m.checkPeerCount(context.Background())
    if err == nil {
        t.Fatal("expected error when below threshold after having been above, got nil")
    }
    if !strings.Contains(err.Error(), "below minimum") {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestPeerCount_EqualPlusTwo_DoesNotFlipFlag(t *testing.T) {
    // If peer count is exactly MinPeers+2, flag should remain false
    min := 5
    rpc := &fakePeerRPC{ret: uint64(min + 2)}
    ep := config.Endpoint{MinPeers: min}
    m := newPeerTestMonitor(t, rpc, ep)

    if err := m.checkPeerCount(context.Background()); err != nil {
        t.Fatalf("unexpected error on boundary: %v", err)
    }
    if m.hasEverBeenAboveMin {
        t.Fatalf("expected hasEverBeenAboveMin=false at boundary, got true")
    }

    // And still should not alert if below since we've never been above
    rpc.ret = uint64(min - 1)
    if err := m.checkPeerCount(context.Background()); err != nil {
        t.Fatalf("expected suppressed error before ever-above, got %v", err)
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
