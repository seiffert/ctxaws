package ctxaws

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"golang.org/x/net/context"
)

// ErrDeadlineWouldExceedBeforeRetry is set as a request's error when a retry is cancelled because
// the backoff interval would exceed the context's deadline.
var ErrDeadlineWouldExceedBeforeRetry = errors.New("deadline would exceed before next retry")

// NewContextAwareRetryer creates a Retryer that honors the given context's deadline.
func NewContextAwareRetryer(ctx context.Context) *Retryer {
	return &Retryer{
		ctx: ctx,
	}
}

// Retryer is an implementation of `request.Retryer` that honors its context's deadline. While the
// delay mechanism is exactly the same as in the `client.DefaultRetryer`, it instructs its clients
// not to retry requests when the context would expire while waiting for the next try. Also requests
// won't be retried when the context already has expired.
type Retryer struct {
	client.DefaultRetryer

	ctx context.Context
}

// ShouldRetry compares the next retry time with the context's deadline and returns false if the
// deadline would occur prior to the next try. If not, the call is passed through to the default
// retryer.
func (r *Retryer) ShouldRetry(req *request.Request) bool {
	if r.ctx.Err() != nil {
		req.Error = r.ctx.Err()
		return false
	}

	if deadline, ok := r.ctx.Deadline(); ok {
		if deadline.Sub(time.Now()) < r.RetryRules(req) {
			req.Error = ErrDeadlineWouldExceedBeforeRetry
			return false
		}
	}

	return r.DefaultRetryer.ShouldRetry(req)
}
