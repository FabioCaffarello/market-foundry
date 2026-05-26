package execution

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/problem"
)

// binanceTestnetBaseURL is the Binance Futures testnet REST endpoint.
const binanceTestnetBaseURL = "https://testnet.binancefuture.com"

// BinanceFuturesTestnetAdapter implements ports.VenuePort for the Binance Futures testnet.
// Scope: market orders only, single symbol, synchronous fills, testnet only.
// Security: credentials are never logged or included in error messages.
type BinanceFuturesTestnetAdapter struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewBinanceFuturesTestnetAdapter creates a venue adapter targeting Binance Futures testnet.
// creds must contain API_KEY and API_SECRET loaded via LoadCredentials.
// submitTimeout controls the HTTP client timeout per request.
func NewBinanceFuturesTestnetAdapter(creds *CredentialSet, submitTimeout time.Duration) *BinanceFuturesTestnetAdapter {
	return &BinanceFuturesTestnetAdapter{
		baseURL:   binanceTestnetBaseURL,
		apiKey:    creds.Get("API_KEY"),
		apiSecret: creds.Get("API_SECRET"),
		httpClient: &http.Client{
			Timeout: submitTimeout,
		},
	}
}

// WithBaseURL overrides the base URL (used for testing with httptest.Server).
func (a *BinanceFuturesTestnetAdapter) WithBaseURL(baseURL string) *BinanceFuturesTestnetAdapter {
	a.baseURL = baseURL
	return a
}

// defaultRequestDeadline is the fallback per-request context deadline
// when the caller does not supply one. Configurable via the HTTP client timeout,
// but this ensures no venue call ever runs without a deadline (EC-3).
const defaultRequestDeadline = 10 * time.Second

// SubmitOrder places a market order on Binance Futures testnet.
// No-action intents (Side=none) are returned immediately without venue interaction.
// Invariant: kill switch and staleness checks happen in the actor layer, not here.
// EC-3: If the incoming context has no deadline, a default deadline is enforced.
func (a *BinanceFuturesTestnetAdapter) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
	// EC-3: Enforce per-request context deadline.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultRequestDeadline)
		defer cancel()
	}

	intent := req.Intent

	// No-action intents: nothing to submit to venue.
	if intent.Side == domainexec.SideNone {
		return ports.VenueOrderReceipt{
			VenueOrderID: fmt.Sprintf("binance-noop-%d", time.Now().UnixNano()),
			Status:       domainexec.StatusAccepted,
			Intent:       intent,
		}, nil
	}

	// Build order parameters.
	side := "BUY"
	if intent.Side == domainexec.SideSell {
		side = "SELL"
	}

	params := url.Values{}
	params.Set("symbol", mapSymbol(intent.VenueSymbol()))
	params.Set("side", side)
	params.Set("type", "MARKET")
	params.Set("quantity", intent.Quantity)
	params.Set("newOrderRespType", "RESULT")
	params.Set("newClientOrderId", ClientOrderID(intent))
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	// Sign the request.
	signature := a.sign(params.Encode())
	params.Set("signature", signature)

	// Build HTTP request.
	endpoint := a.baseURL + "/fapi/v1/order"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "build venue request failed")
	}
	httpReq.Header.Set("X-MBX-APIKEY", a.apiKey)

	// Execute request.
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Unavailable, "venue request failed").MarkRetryable()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		// S322: Mark body-read-failure-after-200 so downstream reconciliation can detect it.
		// Non-retryable because the venue has already accepted the order (HTTP 200 received).
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "read venue response failed").
			WithDetail("body_read_failure_after_200", true).
			WithDetail("client_order_id", ClientOrderID(intent))
	}

	// Handle error responses.
	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp.StatusCode, body)
	}

	// Parse successful response.
	return a.parseOrderResponse(body, intent)
}

// binanceOrderResponse represents the relevant fields from Binance Futures order response.
type binanceOrderResponse struct {
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	Symbol        string `json:"symbol"`
	Status        string `json:"status"`
	Side          string `json:"side"`
	Type          string `json:"type"`
	AvgPrice      string `json:"avgPrice"`
	ExecutedQty   string `json:"executedQty"`
	CumQuote      string `json:"cumQuote"`
	UpdateTime    int64  `json:"updateTime"`
}

// binanceErrorResponse represents a Binance API error.
type binanceErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func (a *BinanceFuturesTestnetAdapter) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(a.apiSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *BinanceFuturesTestnetAdapter) handleErrorResponse(statusCode int, body []byte) (ports.VenueOrderReceipt, *problem.Problem) {
	var errResp binanceErrorResponse
	_ = json.Unmarshal(body, &errResp)

	// Structured details for observability (never contains credentials).
	details := map[string]any{
		"venue_http_status": statusCode,
	}
	if errResp.Code != 0 {
		details["venue_error_code"] = errResp.Code
	}

	// S325: Check venue error code for classification override before HTTP-based routing.
	// Some Binance error codes carry stronger semantic signal than the HTTP status alone.
	// This enrichment is applied ONLY within the 4xx client-error range where the HTTP
	// status is ambiguous but the venue code disambiguates the actual failure class.
	if override, ok := a.classifyByVenueErrorCode(statusCode, errResp.Code, details); ok {
		return ports.VenueOrderReceipt{}, override
	}

	// Classify the error without leaking credentials.
	// Classification follows S308 §2.5 C-FAIL taxonomy (8 failure classes)
	// and S310 §6.2 retryability semantics.
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		// C-FAIL class 1: Authentication — non-retryable.
		return ports.VenueOrderReceipt{}, problem.Newf(problem.InvalidArgument,
			"venue authentication failed (HTTP %d, code %d)", statusCode, errResp.Code).
			WithDetails(details)

	case statusCode == http.StatusTooManyRequests:
		// C-FAIL class 3: Rate limit — retryable.
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue rate limited (HTTP %d)", statusCode).
			WithDetails(details).MarkRetryable()

	case statusCode >= 400 && statusCode < 500:
		// C-FAIL class 2: Client error — non-retryable.
		return ports.VenueOrderReceipt{}, problem.Newf(problem.InvalidArgument,
			"venue rejected order (HTTP %d, code %d): %s", statusCode, errResp.Code, errResp.Message).
			WithDetails(details)

	case statusCode == http.StatusServiceUnavailable:
		// C-FAIL class 4: Venue unavailable — retryable.
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue unavailable (HTTP %d)", statusCode).
			WithDetails(details).MarkRetryable()

	case statusCode == http.StatusBadGateway:
		// C-FAIL class 5: Server error (502 Bad Gateway) — retryable.
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue bad gateway (HTTP %d)", statusCode).
			WithDetails(details).MarkRetryable()

	default:
		// C-FAIL class 5: Server error (5xx catch-all) — retryable.
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue server error (HTTP %d)", statusCode).
			WithDetails(details).MarkRetryable()
	}
}

// classifyByVenueErrorCode applies venue-error-code-aware classification overrides.
// Returns (problem, true) if the venue code overrides the HTTP-based classification,
// or (nil, false) to fall through to the standard HTTP-based classification.
//
// S325: Only three Binance error codes are mapped here — all cases where the HTTP
// status (typically 400 or 418) would produce a non-retryable InvalidArgument, but
// the venue code reveals the actual failure is transient and retryable.
//
// Mapped codes:
//   - -1001: Binance internal error — venue-side transient failure, not a client error.
//   - -1003: Too many requests (IP-level) — rate-limit enforced via IP ban (HTTP 418).
//   - -1015: Too many new orders — order-rate-limit sometimes returned as HTTP 400.
//
// Unmapped codes fall through to HTTP-based classification, which remains the default.
func (a *BinanceFuturesTestnetAdapter) classifyByVenueErrorCode(statusCode, venueCode int, details map[string]any) (*problem.Problem, bool) {
	// Only override within the 4xx range. 401/403 (auth) and 429 (rate limit)
	// are already correctly classified by HTTP status and must not be overridden.
	if statusCode < 400 || statusCode >= 500 {
		return nil, false
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests {
		return nil, false
	}

	switch venueCode {
	case -1001:
		// Binance internal error: venue-side transient failure surfaced as HTTP 400.
		// Reclassify as Unavailable + retryable (was: InvalidArgument, non-retryable).
		details["venue_error_class"] = "venue_internal"
		return problem.Newf(problem.Unavailable,
			"venue internal error (HTTP %d, code %d)", statusCode, venueCode).
			WithDetails(details).MarkRetryable(), true

	case -1003:
		// IP-level rate limit: Binance returns HTTP 418 with code -1003 for IP bans.
		// Reclassify as Unavailable + retryable (was: InvalidArgument, non-retryable).
		details["venue_error_class"] = "ip_rate_limit"
		return problem.Newf(problem.Unavailable,
			"venue IP rate limited (HTTP %d, code %d)", statusCode, venueCode).
			WithDetails(details).MarkRetryable(), true

	case -1015:
		// Order rate limit: Binance sometimes returns HTTP 400 for order rate limits.
		// Reclassify as Unavailable + retryable (was: InvalidArgument, non-retryable).
		details["venue_error_class"] = "order_rate_limit"
		return problem.Newf(problem.Unavailable,
			"venue order rate limited (HTTP %d, code %d)", statusCode, venueCode).
			WithDetails(details).MarkRetryable(), true
	}

	return nil, false
}

func (a *BinanceFuturesTestnetAdapter) parseOrderResponse(body []byte, intent domainexec.ExecutionIntent) (ports.VenueOrderReceipt, *problem.Problem) {
	var resp binanceOrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "parse venue response failed")
	}

	venueOrderID := strconv.FormatInt(resp.OrderID, 10)

	// Map Binance status to domain status.
	status, prob := mapBinanceStatus(resp.Status)
	if prob != nil {
		return ports.VenueOrderReceipt{}, prob
	}

	filled := intent
	filled.Status = status
	filled.FilledQuantity = resp.ExecutedQty

	// Build fill record for filled/partially_filled orders.
	// S428: Fee is "0" because Futures RESULT response does not include commission.
	// CostBasis carries cumQuote (the actual notional value), which was previously
	// misplaced in the Fee field. This separation enables correct cross-segment queries.
	if status == domainexec.StatusFilled || status == domainexec.StatusPartiallyFilled {
		costBasis := "0"
		if resp.CumQuote != "" {
			costBasis = resp.CumQuote
		}

		fillTime := time.Now().UTC()
		if resp.UpdateTime > 0 {
			fillTime = time.UnixMilli(resp.UpdateTime).UTC()
		}

		filled.Fills = []domainexec.FillRecord{
			{
				Price:     resp.AvgPrice,
				Quantity:  resp.ExecutedQty,
				Fee:       "0",
				FeeAsset:  "",
				CostBasis: costBasis,
				FeeSource: domainexec.FeeSourceUnavailable,
				Simulated: false,
				Timestamp: fillTime,
			},
		}
	}

	return ports.VenueOrderReceipt{
		VenueOrderID:  venueOrderID,
		ClientOrderID: ClientOrderID(intent),
		Status:        status,
		Intent:        filled,
	}, nil
}

func mapBinanceStatus(status string) (domainexec.Status, *problem.Problem) {
	switch status {
	case "NEW":
		return domainexec.StatusAccepted, nil
	case "FILLED":
		return domainexec.StatusFilled, nil
	case "PARTIALLY_FILLED":
		return domainexec.StatusPartiallyFilled, nil
	case "CANCELED", "CANCELLED":
		return domainexec.StatusCancelled, nil
	case "REJECTED", "EXPIRED":
		return domainexec.StatusRejected, nil
	default:
		return "", problem.Newf(problem.Internal, "unknown venue status %q", status)
	}
}

// QueryOrder queries the venue for an existing order by client order ID and symbol.
// This is used for post-200 reconciliation: when SubmitOrder received HTTP 200 but
// failed to read the response body, QueryOrder recovers the order status and fills.
//
// Binance API: GET /fapi/v1/order with origClientOrderId parameter.
// Security: same credential handling as SubmitOrder, no secrets in errors.
// EC-3: per-request deadline enforced.
func (a *BinanceFuturesTestnetAdapter) QueryOrder(ctx context.Context, clientOrderID, symbol string) (ports.VenueOrderReceipt, *problem.Problem) {
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

	endpoint := a.baseURL + "/fapi/v1/order"
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

	// Parse the response. We build a synthetic intent with just the instrument for fill parsing.
	// The caller is expected to supply the original intent for full context.
	syntheticIntent := domainexec.ExecutionIntent{Instrument: instrumentFromBinding("binancef", symbol)}
	return a.parseOrderResponse(body, syntheticIntent)
}

// FuturesAccountInfo holds the subset of Binance Futures account data used for
// authenticated connectivity proofs. Read-only surface.
// S441: Introduced for authenticated mainnet proof without order submission.
type FuturesAccountInfo struct {
	CanTrade           bool   `json:"canTrade"`
	FeeTier            int    `json:"feeTier"`
	TotalWalletBalance string `json:"-"`
	AssetCount         int    `json:"-"`
	PositionCount      int    `json:"-"`
	HTTPStatus         int    `json:"-"`
}

// AccountStatus performs an authenticated read-only call to the Binance Futures
// account endpoint (GET /fapi/v2/account). This proves:
//   - API key and secret are valid and correctly signed (HMAC-SHA256)
//   - The mainnet endpoint is reachable and accepts authenticated requests
//   - The account exists and returns permissions and balance metadata
//
// S441: This method is used exclusively for connectivity proofs and soak tests.
// It never submits orders, modifies account state, or triggers any write operation.
func (a *BinanceFuturesTestnetAdapter) AccountStatus(ctx context.Context) (FuturesAccountInfo, *problem.Problem) {
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

	endpoint := a.baseURL + "/fapi/v2/account"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return FuturesAccountInfo{}, problem.Wrap(err, problem.Internal, "build futures account status request failed")
	}
	httpReq.Header.Set("X-MBX-APIKEY", a.apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return FuturesAccountInfo{}, problem.Wrap(err, problem.Unavailable, "futures account status request failed").MarkRetryable()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return FuturesAccountInfo{}, problem.Wrap(err, problem.Internal, "read futures account status response failed")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp binanceErrorResponse
		_ = json.Unmarshal(body, &errResp)
		return FuturesAccountInfo{HTTPStatus: resp.StatusCode}, problem.Newf(problem.InvalidArgument,
			"futures account status failed (HTTP %d, code %d): %s", resp.StatusCode, errResp.Code, errResp.Message)
	}

	var raw struct {
		CanTrade           bool   `json:"canTrade"`
		FeeTier            int    `json:"feeTier"`
		TotalWalletBalance string `json:"totalWalletBalance"`
		Assets             []struct {
			Asset string `json:"asset"`
		} `json:"assets"`
		Positions []struct {
			Symbol string `json:"symbol"`
		} `json:"positions"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return FuturesAccountInfo{}, problem.Wrap(err, problem.Internal, "parse futures account status response failed")
	}

	return FuturesAccountInfo{
		CanTrade:           raw.CanTrade,
		FeeTier:            raw.FeeTier,
		TotalWalletBalance: raw.TotalWalletBalance,
		AssetCount:         len(raw.Assets),
		PositionCount:      len(raw.Positions),
		HTTPStatus:         resp.StatusCode,
	}, nil
}

// mapSymbol normalizes the internal lowercase symbol to Binance's uppercase convention.
func mapSymbol(symbol string) string {
	// Binance expects uppercase: "BTCUSDT" not "btcusdt".
	result := make([]byte, len(symbol))
	for i, c := range []byte(symbol) {
		if c >= 'a' && c <= 'z' {
			result[i] = c - 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
