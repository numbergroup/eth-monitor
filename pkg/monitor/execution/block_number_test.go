package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/numbergroup/eth-monitor/pkg/config"
	"github.com/sirupsen/logrus"
)

// fakeRPC implements the minimal BlockNumber method used by the monitor.
type fakeRPC struct {
	ret uint64
	err error
}

func (f *fakeRPC) BlockNumber(ctx context.Context) (uint64, error) { //nolint:revive // match interface
	return f.ret, f.err
}

func newTestMonitor(t *testing.T, rpc RPCBlockNumber, ep config.Endpoint) *BlockNumberMonitor {
	t.Helper()
	conf := &config.Config{Log: logrus.New()}
	mon, err := NewBlockNumberMonitor(conf, nil, rpc, ep)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	bm, ok := mon.(*BlockNumberMonitor)
	if !ok {
		t.Fatalf("unexpected monitor type: %T", mon)
	}
	return bm
}

func TestCheckNewBlock_FirstIncrease_OK(t *testing.T) {
	rpc := &fakeRPC{ret: 5}
	ep := config.Endpoint{NewBlockMaxDuration: 5 * time.Second}
	m := newTestMonitor(t, rpc, ep)

	if err := m.checkNewBlock(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if m.lastBlockNumber != 5 {
		t.Fatalf("expected lastBlockNumber=5, got %d", m.lastBlockNumber)
	}
	if time.Since(m.lastNewBlockTime) > time.Second {
		t.Fatalf("lastNewBlockTime not updated recently: %v", m.lastNewBlockTime)
	}
}

func TestCheckNewBlock_UnchangedWithinThreshold_OK(t *testing.T) {
	rpc := &fakeRPC{ret: 10}
	ep := config.Endpoint{NewBlockMaxDuration: 2 * time.Second}
	m := newTestMonitor(t, rpc, ep)
	m.lastBlockNumber = 10
	m.lastNewBlockTime = time.Now().Add(-500 * time.Millisecond)

	if err := m.checkNewBlock(t.Context()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheckNewBlock_UnchangedExceedsThreshold_Error(t *testing.T) {
	rpc := &fakeRPC{ret: 10}
	ep := config.Endpoint{NewBlockMaxDuration: 1 * time.Second}
	m := newTestMonitor(t, rpc, ep)
	m.lastBlockNumber = 10
	m.lastNewBlockTime = time.Now().Add(-10 * time.Second)

	err := m.checkNewBlock(t.Context())
	if err == nil {
		t.Fatal("expected error due to no new block, got nil")
	}
	if !strings.Contains(err.Error(), "no new block") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCheckNewBlock_Decrease_Error(t *testing.T) {
	rpc := &fakeRPC{ret: 15}
	ep := config.Endpoint{NewBlockMaxDuration: 5 * time.Second}
	m := newTestMonitor(t, rpc, ep)
	m.lastBlockNumber = 20
	m.lastNewBlockTime = time.Now().Add(-100 * time.Millisecond)

	err := m.checkNewBlock(t.Context())
	if err == nil {
		t.Fatal("expected error on decreased block number, got nil")
	}
	if !strings.Contains(err.Error(), "block number decreased") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if m.lastBlockNumber != 15 {
		t.Fatalf("expected lastBlockNumber updated to 15, got %d", m.lastBlockNumber)
	}
}

func TestCheckNewBlock_RPCError_Propagates(t *testing.T) {
	rpc := &fakeRPC{ret: 0, err: assertErr{}}
	ep := config.Endpoint{NewBlockMaxDuration: 5 * time.Second}
	m := newTestMonitor(t, rpc, ep)

	err := m.checkNewBlock(t.Context())
	if err == nil {
		t.Fatal("expected error from RPC, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get block number") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }

func TestName_Format(t *testing.T) {
	rpc := &fakeRPC{ret: 1}
	ep := config.Endpoint{Name: "example"}
	m := newTestMonitor(t, rpc, ep)

	got := m.Name()
	want := "execution::BlockNumberMonitor::example"
	if got != want {
		t.Fatalf("unexpected name, got %q want %q", got, want)
	}
}
