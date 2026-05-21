package execution

import (
	"context"
	"testing"

	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/problem"
	"internal/shared/settings"
)

// ── S400: SegmentRouter unit tests ───────────────────────────────────

func TestSegmentRouterRoutesFuturesIntentToFuturesAdapter(t *testing.T) {
	router := NewSegmentRouter()
	futuresAdapter := &stubVenueAdapter{label: "futures"}
	router.Register(settings.MarketSegmentFutures, futuresAdapter)

	intent := domainexec.ExecutionIntent{Source: "binancef", Symbol: "btcusdt"}
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}
	if futuresAdapter.callCount != 1 {
		t.Fatalf("expected 1 call to futures adapter, got %d", futuresAdapter.callCount)
	}
}

func TestSegmentRouterRoutesSpotIntentToSpotAdapter(t *testing.T) {
	router := NewSegmentRouter()
	spotAdapter := &stubVenueAdapter{label: "spot"}
	router.Register(settings.MarketSegmentSpot, spotAdapter)

	intent := domainexec.ExecutionIntent{Source: "binances", Symbol: "ethusdt"}
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}
	if spotAdapter.callCount != 1 {
		t.Fatalf("expected 1 call to spot adapter, got %d", spotAdapter.callCount)
	}
}

func TestSegmentRouterMultiSegmentIsolation(t *testing.T) {
	router := NewSegmentRouter()
	futuresAdapter := &stubVenueAdapter{label: "futures"}
	spotAdapter := &stubVenueAdapter{label: "spot"}
	router.Register(settings.MarketSegmentFutures, futuresAdapter)
	router.Register(settings.MarketSegmentSpot, spotAdapter)

	// Route futures intent.
	futuresIntent := domainexec.ExecutionIntent{Source: "binancef", Symbol: "btcusdt"}
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: futuresIntent})
	if prob != nil {
		t.Fatalf("futures routing failed: %s", prob.Message)
	}

	// Route spot intent.
	spotIntent := domainexec.ExecutionIntent{Source: "binances", Symbol: "ethusdt"}
	_, prob = router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: spotIntent})
	if prob != nil {
		t.Fatalf("spot routing failed: %s", prob.Message)
	}

	if futuresAdapter.callCount != 1 {
		t.Fatalf("futures adapter: expected 1 call, got %d", futuresAdapter.callCount)
	}
	if spotAdapter.callCount != 1 {
		t.Fatalf("spot adapter: expected 1 call, got %d", spotAdapter.callCount)
	}
}

func TestSegmentRouterRejectsUnknownSource(t *testing.T) {
	router := NewSegmentRouter()
	router.Register(settings.MarketSegmentFutures, &stubVenueAdapter{label: "futures"})

	intent := domainexec.ExecutionIntent{Source: "unknown_exchange", Symbol: "btcusdt"}
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
}

func TestSegmentRouterRejectsSourceWithNoRegisteredAdapter(t *testing.T) {
	router := NewSegmentRouter()
	// Register futures but not spot.
	router.Register(settings.MarketSegmentFutures, &stubVenueAdapter{label: "futures"})

	intent := domainexec.ExecutionIntent{Source: "binances", Symbol: "btcusdt"}
	_, prob := router.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob == nil {
		t.Fatal("expected error for unregistered spot segment, got nil")
	}
}

func TestSegmentRouterSegmentCount(t *testing.T) {
	router := NewSegmentRouter()
	if router.SegmentCount() != 0 {
		t.Fatalf("empty router: expected 0 segments, got %d", router.SegmentCount())
	}

	router.Register(settings.MarketSegmentFutures, &stubVenueAdapter{label: "futures"})
	if router.SegmentCount() != 1 {
		t.Fatalf("expected 1 segment, got %d", router.SegmentCount())
	}

	router.Register(settings.MarketSegmentSpot, &stubVenueAdapter{label: "spot"})
	if router.SegmentCount() != 2 {
		t.Fatalf("expected 2 segments, got %d", router.SegmentCount())
	}
}

func TestSegmentRouterHasSegment(t *testing.T) {
	router := NewSegmentRouter()
	router.Register(settings.MarketSegmentFutures, &stubVenueAdapter{label: "futures"})

	if !router.HasSegment(settings.MarketSegmentFutures) {
		t.Fatal("expected HasSegment(futures) = true")
	}
	if router.HasSegment(settings.MarketSegmentSpot) {
		t.Fatal("expected HasSegment(spot) = false")
	}
}

// stubVenueAdapter is a test double that counts calls.
type stubVenueAdapter struct {
	label     string
	callCount int
}

func (s *stubVenueAdapter) SubmitOrder(_ context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	s.callCount++
	return ports.VenueOrderReceipt{
		VenueOrderID: s.label + "-order-1",
		Status:       domainexec.StatusFilled,
		Intent:       req.Intent,
	}, nil
}
