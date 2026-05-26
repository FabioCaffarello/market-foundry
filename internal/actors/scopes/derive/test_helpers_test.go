package derive

import (
	"sync"
	"testing"
	"time"

	"internal/domain/instrument"
	"internal/domain/observation"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

func btcUSDTPerp() instrument.CanonicalInstrument {
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		panic("test setup: failed to build canonical BTC/USDT-perpetual: " + prob.Message)
	}
	return inst
}

// msgCollector is a lightweight actor that records all non-lifecycle messages.
// Used in tests as a stand-in for publisher and scope PIDs.
type msgCollector struct {
	mu   sync.Mutex
	msgs []any
	ch   chan struct{}
}

func newMsgCollector() *msgCollector {
	return &msgCollector{ch: make(chan struct{}, 100)}
}

func (c *msgCollector) producer() actor.Producer {
	return func() actor.Receiver { return c }
}

func (c *msgCollector) Receive(ctx *actor.Context) {
	switch ctx.Message().(type) {
	case actor.Started, actor.Stopped, actor.Initialized:
		return
	default:
		c.mu.Lock()
		c.msgs = append(c.msgs, ctx.Message())
		c.mu.Unlock()
		c.ch <- struct{}{}
	}
}

// waitFor blocks until count messages have been collected or timeout expires.
func (c *msgCollector) waitFor(t *testing.T, count int, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for i := 0; i < count; i++ {
		select {
		case <-c.ch:
		case <-deadline:
			c.mu.Lock()
			got := len(c.msgs)
			c.mu.Unlock()
			t.Fatalf("timeout waiting for message %d/%d (got %d so far)", i+1, count, got)
		}
	}
}

// messages returns a snapshot of all collected messages.
func (c *msgCollector) messages() []any {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]any, len(c.msgs))
	copy(out, c.msgs)
	return out
}

// count returns the current number of collected messages.
func (c *msgCollector) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.msgs)
}

// --- trade factory helpers ---

// windowBase returns a time aligned to a 60-second window boundary.
func windowBase() time.Time {
	return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
}

// makeTrade creates a valid ObservationTrade at the given offset from base.
func makeTrade(base time.Time, offset time.Duration, price, qty string) observation.TradeReceivedEvent {
	ts := base.Add(offset)
	return observation.TradeReceivedEvent{
		Metadata: events.NewMetadata(),
		Trade: observation.ObservationTrade{
			Source:     "binancef",
			Instrument: btcUSDTPerp(),
			Price:      price,
			Quantity:   qty,
			TradeID:    ts.Format("150405.000"),
			Timestamp:  ts,
		},
	}
}

// makeTradeWithSide creates a trade with the BuyerMaker flag set.
func makeTradeWithSide(base time.Time, offset time.Duration, price, qty string, buyerMaker bool) observation.TradeReceivedEvent {
	e := makeTrade(base, offset, price, qty)
	e.Trade.BuyerMaker = buyerMaker
	return e
}

// newTestEngine creates a Hollywood engine for testing. Caller must call e.Poison(pids...) when done.
func newTestEngine(t *testing.T) *actor.Engine {
	t.Helper()
	e, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	return e
}
