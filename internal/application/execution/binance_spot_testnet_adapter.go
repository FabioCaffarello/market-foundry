package execution

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	appingest "internal/application/ingest"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/problem"
)

// binanceSpotTestnetBaseURL is the Binance Spot testnet REST endpoint.
const binanceSpotTestnetBaseURL = "https://testnet.binance.vision"

// BinanceSpotTestnetAdapter implements ports.VenuePort for the Binance Spot testnet.
// Scope: market orders only, single symbol, synchronous fills, testnet only.
// Security: credentials are never logged or included in error messages.
//
// S392: Spot adapter is structurally parallel to Futures but with Spot-specific
// response parsing. Key differences:
//   - Base URL: testnet.binance.vision (vs testnet.binancefuture.com)
//   - API path: /api/v3/order (vs /fapi/v1/order)
//   - Response: fills[] array with per-leg price/qty/commission (vs avgPrice/cumQuote)
//   - No top-level avgPrice — computed as weighted average from fills
type BinanceSpotTestnetAdapter struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewBinanceSpotTestnetAdapter creates a venue adapter targeting Binance Spot testnet.
// creds must contain API_KEY and API_SECRET loaded via LoadCredentials.
// submitTimeout controls the HTTP client timeout per request.
func NewBinanceSpotTestnetAdapter(creds *CredentialSet, submitTimeout time.Duration) *BinanceSpotTestnetAdapter {
	return &BinanceSpotTestnetAdapter{
		baseURL:   binanceSpotTestnetBaseURL,
		apiKey:    creds.Get("API_KEY"),
		apiSecret: creds.Get("API_SECRET"),
		httpClient: &http.Client{
			Timeout: submitTimeout,
		},
	}
}

// WithBaseURL overrides the base URL (used for testing with httptest.Server).
func (a *BinanceSpotTestnetAdapter) WithBaseURL(baseURL string) *BinanceSpotTestnetAdapter {
	a.baseURL = baseURL
	return a
}

// SubmitOrder places a market order on Binance Spot testnet.
// No-action intents (Side=none) are returned immediately without venue interaction.
// EC-3: If the incoming context has no deadline, a default deadline is enforced.
func (a *BinanceSpotTestnetAdapter) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultRequestDeadline)
		defer cancel()
	}

	intent := req.Intent

	if intent.Side == domainexec.SideNone {
		return ports.VenueOrderReceipt{
			VenueOrderID: fmt.Sprintf("binance-spot-noop-%d", time.Now().UnixNano()),
			Status:       domainexec.StatusAccepted,
			Intent:       intent,
		}, nil
	}

	side := "BUY"
	if intent.Side == domainexec.SideSell {
		side = "SELL"
	}

	params := url.Values{}
	params.Set("symbol", mapSymbol(intent.VenueSymbol()))
	params.Set("side", side)
	params.Set("type", "MARKET")
	params.Set("quantity", intent.Quantity)
	params.Set("newOrderRespType", "FULL")
	params.Set("newClientOrderId", ClientOrderID(intent))
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	signature := a.sign(params.Encode())
	params.Set("signature", signature)

	endpoint := a.baseURL + "/api/v3/order"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "build venue request failed")
	}
	httpReq.Header.Set("X-MBX-APIKEY", a.apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Unavailable, "venue request failed").MarkRetryable()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "read venue response failed").
			WithDetail("body_read_failure_after_200", true).
			WithDetail("client_order_id", ClientOrderID(intent))
	}

	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp.StatusCode, body)
	}

	return a.parseOrderResponse(body, intent)
}

// binanceSpotOrderResponse represents the relevant fields from Binance Spot order response.
// Key differences from Futures:
//   - No avgPrice — computed from fills
//   - cummulativeQuoteQty instead of cumQuote
//   - fills[] array with per-leg price, qty, commission, commissionAsset
type binanceSpotOrderResponse struct {
	OrderID             int64                  `json:"orderId"`
	ClientOrderID       string                 `json:"clientOrderId"`
	Symbol              string                 `json:"symbol"`
	Status              string                 `json:"status"`
	Side                string                 `json:"side"`
	Type                string                 `json:"type"`
	ExecutedQty         string                 `json:"executedQty"`
	CummulativeQuoteQty string                 `json:"cummulativeQuoteQty"`
	TransactTime        int64                  `json:"transactTime"`
	Fills               []binanceSpotFillEntry `json:"fills"`
}

// binanceSpotFillEntry represents a single fill leg from the Spot order response.
type binanceSpotFillEntry struct {
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
}

func (a *BinanceSpotTestnetAdapter) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(a.apiSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *BinanceSpotTestnetAdapter) handleErrorResponse(statusCode int, body []byte) (ports.VenueOrderReceipt, *problem.Problem) {
	var errResp binanceErrorResponse
	_ = json.Unmarshal(body, &errResp)

	details := map[string]any{
		"venue_http_status": statusCode,
	}
	if errResp.Code != 0 {
		details["venue_error_code"] = errResp.Code
	}

	if override, ok := a.classifyByVenueErrorCode(statusCode, errResp.Code, details); ok {
		return ports.VenueOrderReceipt{}, override
	}

	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.InvalidArgument,
			"venue authentication failed (HTTP %d, code %d)", statusCode, errResp.Code).
			WithDetails(details)

	case statusCode == http.StatusTooManyRequests:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue rate limited (HTTP %d)", statusCode).
			WithDetails(details).MarkRetryable()

	case statusCode >= 400 && statusCode < 500:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.InvalidArgument,
			"venue rejected order (HTTP %d, code %d): %s", statusCode, errResp.Code, errResp.Message).
			WithDetails(details)

	case statusCode == http.StatusServiceUnavailable:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue unavailable (HTTP %d)", statusCode).
			WithDetails(details).MarkRetryable()

	case statusCode == http.StatusBadGateway:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue bad gateway (HTTP %d)", statusCode).
			WithDetails(details).MarkRetryable()

	default:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue server error (HTTP %d)", statusCode).
			WithDetails(details).MarkRetryable()
	}
}

func (a *BinanceSpotTestnetAdapter) classifyByVenueErrorCode(statusCode, venueCode int, details map[string]any) (*problem.Problem, bool) {
	if statusCode < 400 || statusCode >= 500 {
		return nil, false
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests {
		return nil, false
	}

	switch venueCode {
	case -1001:
		details["venue_error_class"] = "venue_internal"
		return problem.Newf(problem.Unavailable,
			"venue internal error (HTTP %d, code %d)", statusCode, venueCode).
			WithDetails(details).MarkRetryable(), true
	case -1003:
		details["venue_error_class"] = "ip_rate_limit"
		return problem.Newf(problem.Unavailable,
			"venue IP rate limited (HTTP %d, code %d)", statusCode, venueCode).
			WithDetails(details).MarkRetryable(), true
	case -1015:
		details["venue_error_class"] = "order_rate_limit"
		return problem.Newf(problem.Unavailable,
			"venue order rate limited (HTTP %d, code %d)", statusCode, venueCode).
			WithDetails(details).MarkRetryable(), true
	}

	return nil, false
}

func (a *BinanceSpotTestnetAdapter) parseOrderResponse(body []byte, intent domainexec.ExecutionIntent) (ports.VenueOrderReceipt, *problem.Problem) {
	var resp binanceSpotOrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "parse venue response failed")
	}

	venueOrderID := strconv.FormatInt(resp.OrderID, 10)

	status, prob := mapBinanceStatus(resp.Status)
	if prob != nil {
		return ports.VenueOrderReceipt{}, prob
	}

	filled := intent
	filled.Status = status
	filled.FilledQuantity = resp.ExecutedQty

	// S428: Spot fills carry real commission and commissionAsset from the fills[] array.
	// CostBasis = cummulativeQuoteQty (total notional value).
	// Fee = aggregated commission across all fill legs.
	// FeeAsset = commissionAsset from the first fill leg (uniform within a single order).
	if status == domainexec.StatusFilled || status == domainexec.StatusPartiallyFilled {
		fillTime := time.Now().UTC()
		if resp.TransactTime > 0 {
			fillTime = time.UnixMilli(resp.TransactTime).UTC()
		}

		costBasis := "0"
		if resp.CummulativeQuoteQty != "" {
			costBasis = resp.CummulativeQuoteQty
		}

		if len(resp.Fills) > 0 {
			avgPrice, totalFee, feeAsset, _ := computeSpotFillAggregates(resp.Fills)
			filled.Fills = []domainexec.FillRecord{
				{
					Price:     avgPrice,
					Quantity:  resp.ExecutedQty,
					Fee:       totalFee,
					FeeAsset:  feeAsset,
					CostBasis: costBasis,
					FeeSource: domainexec.FeeSourceVenue,
					Simulated: false,
					Timestamp: fillTime,
				},
			}
		} else {
			// Fallback: no fills array (shouldn't happen with FULL response type).
			// S499: FeeSourceFallback signals this is an unexpected code path.
			filled.Fills = []domainexec.FillRecord{
				{
					Price:     "0",
					Quantity:  resp.ExecutedQty,
					Fee:       "0",
					CostBasis: costBasis,
					FeeSource: domainexec.FeeSourceFallback,
					Simulated: false,
					Timestamp: fillTime,
				},
			}
		}
	}

	return ports.VenueOrderReceipt{
		VenueOrderID:  venueOrderID,
		ClientOrderID: ClientOrderID(intent),
		Status:        status,
		Intent:        filled,
	}, nil
}

// computeSpotFillAggregates computes weighted average price, total commission, and
// fee asset from Binance Spot per-leg fills. Returns (avgPrice, totalFee, feeAsset, mixed).
// Uses 8 decimal places (Binance standard precision) to avoid floating-point noise.
// S428: feeAsset is taken from the first fill leg — Binance uses a uniform commission
// asset within a single market order.
// S499: mixed is true when fills have different CommissionAssets (non-uniform).
func computeSpotFillAggregates(fills []binanceSpotFillEntry) (string, string, string, bool) {
	var totalQty, totalQuote, totalFee float64
	feeAsset := ""
	mixed := false

	for _, f := range fills {
		price, _ := strconv.ParseFloat(f.Price, 64)
		qty, _ := strconv.ParseFloat(f.Qty, 64)
		commission, _ := strconv.ParseFloat(f.Commission, 64)
		totalQty += qty
		totalQuote += price * qty
		totalFee += commission
		if feeAsset == "" && f.CommissionAsset != "" {
			feeAsset = f.CommissionAsset
		} else if feeAsset != "" && f.CommissionAsset != "" && f.CommissionAsset != feeAsset {
			mixed = true
		}
	}

	avgPrice := "0"
	if totalQty > 0 {
		avgPrice = strconv.FormatFloat(totalQuote/totalQty, 'f', 8, 64)
	}

	return trimTrailingZeros(avgPrice), trimTrailingZeros(strconv.FormatFloat(totalFee, 'f', 8, 64)), feeAsset, mixed
}

// trimTrailingZeros removes trailing zeros from a decimal string.
// "65430.00000000" → "65430", "0.00030000" → "0.0003".
func trimTrailingZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// QueryOrder queries the venue for an existing order by client order ID and symbol.
// Binance Spot API: GET /api/v3/order with origClientOrderId parameter.
func (a *BinanceSpotTestnetAdapter) QueryOrder(ctx context.Context, clientOrderID, symbol string) (ports.VenueOrderReceipt, *problem.Problem) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultRequestDeadline)
		defer cancel()
	}

	params := url.Values{}
	params.Set("symbol", mapSymbol(symbol))
	params.Set("origClientOrderId", clientOrderID)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	signature := a.sign(params.Encode())
	params.Set("signature", signature)

	endpoint := a.baseURL + "/api/v3/order"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "build query request failed")
	}
	httpReq.Header.Set("X-MBX-APIKEY", a.apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Unavailable, "query order request failed").MarkRetryable()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "read query response failed")
	}

	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp.StatusCode, body)
	}

	// H-6.c.2 commit 4: reconstruct Instrument via the canonical
	// BindingTarget boundary helper with warn-and-emit-zero fallback
	// — same error-handling pattern as the composite_reader sites
	// (commit 2). Spot venue identity is hardcoded "binances";
	// reconstruction failure only occurs on symbol-parsing edge cases
	// (non-USDT or empty), which are not in the current production
	// path. See PROGRAM-0004 H-6.f scope notes for the candidate
	// port-signature refactor that eliminates this reconstruction
	// entirely.
	inst, instErr := appingest.BindingTarget{Source: "binances", Symbol: symbol}.Instrument()
	if instErr != nil {
		slog.Default().Warn("instrument reconstruction failed in spot testnet adapter; emitting zero instrument",
			"source", "binances",
			"symbol", symbol,
			"error", instErr,
		)
	}
	syntheticIntent := domainexec.ExecutionIntent{Instrument: inst}
	return a.parseOrderResponse(body, syntheticIntent)
}

// AccountInfo holds the subset of Binance account data used for authenticated
// connectivity proofs. This is a read-only surface — no write operations.
// S441: Introduced for authenticated mainnet proof without order submission.
type AccountInfo struct {
	CanTrade     bool   `json:"canTrade"`
	CanWithdraw  bool   `json:"canWithdraw"`
	AccountType  string `json:"accountType"`
	ServerTimeMs int64  `json:"-"`
	BalanceCount int    `json:"-"`
	HTTPStatus   int    `json:"-"`
}

// AccountStatus performs an authenticated read-only call to the Binance Spot
// account endpoint (GET /api/v3/account). This proves:
//   - API key and secret are valid and correctly signed (HMAC-SHA256)
//   - The mainnet endpoint is reachable and accepts authenticated requests
//   - The account exists and returns permissions metadata
//
// S441: This method is used exclusively for connectivity proofs and soak tests.
// It never submits orders, modifies account state, or triggers any write operation.
func (a *BinanceSpotTestnetAdapter) AccountStatus(ctx context.Context) (AccountInfo, *problem.Problem) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultRequestDeadline)
		defer cancel()
	}

	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	signature := a.sign(params.Encode())
	params.Set("signature", signature)

	endpoint := a.baseURL + "/api/v3/account"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return AccountInfo{}, problem.Wrap(err, problem.Internal, "build account status request failed")
	}
	httpReq.Header.Set("X-MBX-APIKEY", a.apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return AccountInfo{}, problem.Wrap(err, problem.Unavailable, "account status request failed").MarkRetryable()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return AccountInfo{}, problem.Wrap(err, problem.Internal, "read account status response failed")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp binanceErrorResponse
		_ = json.Unmarshal(body, &errResp)
		return AccountInfo{HTTPStatus: resp.StatusCode}, problem.Newf(problem.InvalidArgument,
			"account status failed (HTTP %d, code %d): %s", resp.StatusCode, errResp.Code, errResp.Message)
	}

	var raw struct {
		CanTrade    bool   `json:"canTrade"`
		CanWithdraw bool   `json:"canWithdraw"`
		AccountType string `json:"accountType"`
		ServerTime  int64  `json:"updateTime"`
		Balances    []struct {
			Asset string `json:"asset"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return AccountInfo{}, problem.Wrap(err, problem.Internal, "parse account status response failed")
	}

	return AccountInfo{
		CanTrade:     raw.CanTrade,
		CanWithdraw:  raw.CanWithdraw,
		AccountType:  raw.AccountType,
		ServerTimeMs: raw.ServerTime,
		BalanceCount: len(raw.Balances),
		HTTPStatus:   resp.StatusCode,
	}, nil
}
