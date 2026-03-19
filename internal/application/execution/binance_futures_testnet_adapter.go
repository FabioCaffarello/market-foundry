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

// SubmitOrder places a market order on Binance Futures testnet.
// No-action intents (Side=none) are returned immediately without venue interaction.
// Invariant: kill switch and staleness checks happen in the actor layer, not here.
func (a *BinanceFuturesTestnetAdapter) SubmitOrder(ctx context.Context, req ports.VenueOrderRequest) (ports.VenueOrderReceipt, *problem.Problem) {
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
	params.Set("symbol", mapSymbol(intent.Symbol))
	params.Set("side", side)
	params.Set("type", "MARKET")
	params.Set("quantity", intent.Quantity)
	params.Set("newOrderRespType", "RESULT")
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
		return ports.VenueOrderReceipt{}, problem.Wrap(err, problem.Internal, "read venue response failed")
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
	OrderID       int64   `json:"orderId"`
	Symbol        string  `json:"symbol"`
	Status        string  `json:"status"`
	Side          string  `json:"side"`
	Type          string  `json:"type"`
	AvgPrice      string  `json:"avgPrice"`
	ExecutedQty   string  `json:"executedQty"`
	CumQuote      string  `json:"cumQuote"`
	UpdateTime    int64   `json:"updateTime"`
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

	// Classify the error without leaking credentials.
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.InvalidArgument,
			"venue authentication failed (HTTP %d, code %d)", statusCode, errResp.Code)

	case statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue rate limited or unavailable (HTTP %d)", statusCode).MarkRetryable()

	case statusCode >= 400 && statusCode < 500:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.InvalidArgument,
			"venue rejected order (HTTP %d, code %d): %s", statusCode, errResp.Code, errResp.Message)

	default:
		return ports.VenueOrderReceipt{}, problem.Newf(problem.Unavailable,
			"venue server error (HTTP %d)", statusCode).MarkRetryable()
	}
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
	if status == domainexec.StatusFilled || status == domainexec.StatusPartiallyFilled {
		fee := "0"
		if resp.CumQuote != "" && resp.ExecutedQty != "" {
			fee = resp.CumQuote // cumulative quote as fee proxy (commissions come from separate endpoint)
		}

		fillTime := time.Now().UTC()
		if resp.UpdateTime > 0 {
			fillTime = time.UnixMilli(resp.UpdateTime).UTC()
		}

		filled.Fills = []domainexec.FillRecord{
			{
				Price:     resp.AvgPrice,
				Quantity:  resp.ExecutedQty,
				Fee:       fee,
				Simulated: false,
				Timestamp: fillTime,
			},
		}
	}

	return ports.VenueOrderReceipt{
		VenueOrderID: venueOrderID,
		Status:       status,
		Intent:       filled,
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
