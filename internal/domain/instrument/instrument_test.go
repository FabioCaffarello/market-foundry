package instrument_test

import (
	"testing"

	"internal/domain/instrument"
)

// ── BaseAsset / QuoteAsset ────────────────────────────────────────

func TestNewBaseAsset_Uppercases(t *testing.T) {
	b, prob := instrument.NewBaseAsset("btc")
	if prob != nil {
		t.Fatalf("NewBaseAsset: %v", prob)
	}
	if b != "BTC" {
		t.Errorf("got %q, want %q", b, "BTC")
	}
}

func TestNewBaseAsset_TrimsWhitespace(t *testing.T) {
	b, prob := instrument.NewBaseAsset("  eth  ")
	if prob != nil {
		t.Fatalf("NewBaseAsset: %v", prob)
	}
	if b != "ETH" {
		t.Errorf("got %q, want %q", b, "ETH")
	}
}

func TestNewBaseAsset_RejectsEmpty(t *testing.T) {
	if _, prob := instrument.NewBaseAsset(""); prob == nil {
		t.Fatal("expected error on empty input")
	}
	if _, prob := instrument.NewBaseAsset("   "); prob == nil {
		t.Fatal("expected error on whitespace-only input")
	}
}

func TestNewBaseAsset_AcceptsDigits(t *testing.T) {
	b, prob := instrument.NewBaseAsset("1000PEPE")
	if prob != nil {
		t.Fatalf("NewBaseAsset: %v", prob)
	}
	if b != "1000PEPE" {
		t.Errorf("got %q, want %q", b, "1000PEPE")
	}
}

func TestNewBaseAsset_RejectsDisallowedChars(t *testing.T) {
	cases := []string{"BTC/USDT", "BTC-USDT", "BTC.USDT", "BTC USDT", "btc!"}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, prob := instrument.NewBaseAsset(c); prob == nil {
				t.Errorf("expected error on %q", c)
			}
		})
	}
}

func TestNewBaseAsset_RejectsTooLong(t *testing.T) {
	if _, prob := instrument.NewBaseAsset("ABCDEFGHIJKLMNOPQ"); prob == nil {
		t.Fatal("expected error on 17-char ticker")
	}
}

func TestNewQuoteAsset_ParityWithBase(t *testing.T) {
	q, prob := instrument.NewQuoteAsset("usdt")
	if prob != nil {
		t.Fatalf("NewQuoteAsset: %v", prob)
	}
	if q != "USDT" {
		t.Errorf("got %q, want %q", q, "USDT")
	}
}

// ── Venue ────────────────────────────────────────────────────────

func TestVenue_ValidEnumeration(t *testing.T) {
	if !instrument.ValidVenue(instrument.VenueBinance) {
		t.Error("VenueBinance should be valid")
	}
	if !instrument.ValidVenue(instrument.VenueBinanceFutures) {
		t.Error("VenueBinanceFutures should be valid")
	}
	// H-7.b: Bybit family ships.
	if !instrument.ValidVenue(instrument.VenueBybit) {
		t.Error("VenueBybit should be valid")
	}
	if !instrument.ValidVenue(instrument.VenueBybitFutures) {
		t.Error("VenueBybitFutures should be valid")
	}
}

func TestVenue_InvalidRejected(t *testing.T) {
	// "bybit" left this list in H-7.b when its adapter shipped;
	// "coinbase" stays invalid until its own adapter lands.
	cases := []instrument.Venue{"coinbase", "BINANCE", "BYBIT", "", "unknown"}
	for _, v := range cases {
		t.Run(v.String(), func(t *testing.T) {
			if instrument.ValidVenue(v) {
				t.Errorf("ValidVenue(%q) returned true; want false", v)
			}
			if v.Validate() == nil {
				t.Errorf("%q.Validate() returned nil; want problem", v)
			}
		})
	}
}

// ── ContractType ─────────────────────────────────────────────────

func TestContractType_ValidEnumeration(t *testing.T) {
	cases := []instrument.ContractType{
		instrument.ContractSpot,
		instrument.ContractUSDTFutures,
		instrument.ContractCoinFutures,
		instrument.ContractPerpetual,
	}
	for _, c := range cases {
		t.Run(c.String(), func(t *testing.T) {
			if !instrument.ValidContractType(c) {
				t.Errorf("%q should be valid", c)
			}
			if c.Validate() != nil {
				t.Errorf("%q.Validate() returned problem; want nil", c)
			}
		})
	}
}

func TestContractType_InvalidRejected(t *testing.T) {
	cases := []instrument.ContractType{"SPOT", "futures", "", "options"}
	for _, c := range cases {
		t.Run(c.String(), func(t *testing.T) {
			if instrument.ValidContractType(c) {
				t.Errorf("ValidContractType(%q) returned true; want false", c)
			}
		})
	}
}

// ── CanonicalInstrument ──────────────────────────────────────────

func TestNew_ProducesValidInstrument(t *testing.T) {
	c, prob := instrument.New("btc", "usdt", instrument.ContractSpot)
	if prob != nil {
		t.Fatalf("New: %v", prob)
	}
	if c.Base != "BTC" || c.Quote != "USDT" || c.Contract != instrument.ContractSpot {
		t.Errorf("got %+v", c)
	}
}

func TestNew_RejectsInvalidBase(t *testing.T) {
	if _, prob := instrument.New("", "usdt", instrument.ContractSpot); prob == nil {
		t.Fatal("expected problem on empty base")
	}
}

func TestNew_RejectsInvalidQuote(t *testing.T) {
	if _, prob := instrument.New("btc", "", instrument.ContractSpot); prob == nil {
		t.Fatal("expected problem on empty quote")
	}
}

func TestNew_RejectsInvalidContract(t *testing.T) {
	if _, prob := instrument.New("btc", "usdt", instrument.ContractType("bogus")); prob == nil {
		t.Fatal("expected problem on bogus contract type")
	}
}

func TestSymbol_Format(t *testing.T) {
	cases := []struct {
		base     string
		quote    string
		contract instrument.ContractType
		want     string
	}{
		{"BTC", "USDT", instrument.ContractSpot, "BTC/USDT-spot"},
		{"ETH", "USDT", instrument.ContractPerpetual, "ETH/USDT-perpetual"},
		{"BTC", "USD", instrument.ContractCoinFutures, "BTC/USD-coinfutures"},
		{"BTC", "USDT", instrument.ContractUSDTFutures, "BTC/USDT-usdtfutures"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			c, prob := instrument.New(tc.base, tc.quote, tc.contract)
			if prob != nil {
				t.Fatalf("New: %v", prob)
			}
			if got := c.Symbol(); got != tc.want {
				t.Errorf("Symbol() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	var zero instrument.CanonicalInstrument
	if !zero.IsZero() {
		t.Error("zero value should report IsZero() = true")
	}
	c, _ := instrument.New("btc", "usdt", instrument.ContractSpot)
	if c.IsZero() {
		t.Error("constructed instrument should report IsZero() = false")
	}
}

func TestValidate_ZeroValueRejected(t *testing.T) {
	var zero instrument.CanonicalInstrument
	if zero.Validate() == nil {
		t.Error("zero value Validate() returned nil; want problem")
	}
}

// ── FromSymbol round-trip ────────────────────────────────────────

func TestFromSymbol_RoundTrip(t *testing.T) {
	cases := []string{
		"BTC/USDT-spot",
		"ETH/USDT-perpetual",
		"BTC/USD-coinfutures",
		"BTC/USDT-usdtfutures",
		"1000PEPE/USDT-perpetual",
	}
	for _, want := range cases {
		t.Run(want, func(t *testing.T) {
			c, prob := instrument.FromSymbol(want)
			if prob != nil {
				t.Fatalf("FromSymbol(%q): %v", want, prob)
			}
			if got := c.Symbol(); got != want {
				t.Errorf("round-trip: parsed %q, re-serialized %q", want, got)
			}
		})
	}
}

func TestFromSymbol_RejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"BTC",
		"BTC/USDT",       // missing contract
		"BTC-spot",       // missing /
		"BTCUSDT-spot",   // missing /
		"/USDT-spot",     // empty base
		"BTC/-spot",      // empty quote
		"BTC/USDT-",      // empty contract
		"-BTC/USDT-spot", // misplaced separators
		"BTC/USDT-bogus", // invalid contract type
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, prob := instrument.FromSymbol(in); prob == nil {
				t.Errorf("FromSymbol(%q): expected problem, got nil", in)
			}
		})
	}
}

func TestFromSymbol_NormalizesLowercaseAssets(t *testing.T) {
	// FromSymbol delegates to New, which normalizes (uppercases)
	// asset components. This is generous behaviour: the canonical
	// form is always uppercase per Symbol(), but a slightly
	// lowercase input round-trips cleanly to the canonical form.
	c, prob := instrument.FromSymbol("btc/usdt-spot")
	if prob != nil {
		t.Fatalf("FromSymbol: %v", prob)
	}
	if got := c.Symbol(); got != "BTC/USDT-spot" {
		t.Errorf("normalized to %q, want %q", got, "BTC/USDT-spot")
	}
}

func TestFromSymbol_TrimsWhitespace(t *testing.T) {
	c, prob := instrument.FromSymbol("  BTC/USDT-spot  ")
	if prob != nil {
		t.Fatalf("FromSymbol: %v", prob)
	}
	if c.Symbol() != "BTC/USDT-spot" {
		t.Errorf("got %q", c.Symbol())
	}
}
