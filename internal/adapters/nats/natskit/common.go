package natskit

import "time"

// DefaultSetupTimeout is the timeout used for JetStream setup operations.
const DefaultSetupTimeout = 10 * time.Second

// DefaultRequestTimeout is the bound applied to request/reply handler
// dispatch when no per-responder override is configured. NATS callbacks
// do not carry a Go context, so responders must construct their own.
const DefaultRequestTimeout = 5 * time.Second

// ErrorType identifies the kind of NATS error for metrics.
type ErrorType string

const (
	ErrCreateStream     ErrorType = "create_stream"
	ErrCreateConsumer   ErrorType = "create_consumer"
	ErrStreamNotFound   ErrorType = "stream_404"
	ErrConsumerNotFound ErrorType = "consumer_404"
	ErrMsgNotFound      ErrorType = "msg_404"
	ErrConsumeHandler   ErrorType = "consume_handler"
	ErrStartConsumer    ErrorType = "start_consumer"
	ErrDecode           ErrorType = "decode_error"
)

// ReportError asynchronously reports an error via the given handler.
func ReportError(err error, et ErrorType, h func(error, string)) {
	if err == nil {
		return
	}

	if h == nil {
		return
	}

	go h(err, string(et))
}
