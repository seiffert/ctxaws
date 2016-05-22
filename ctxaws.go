package ctxaws

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/corehandlers"
	"github.com/aws/aws-sdk-go/aws/request"
	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// InContext performs an AWS request and takes care that it does not exceed the context's deadline.
// It replaces the request's send handler with an adapted version. The original was taken from
// https://github.com/aws/aws-sdk-go/blob/7761b2cb776ce24b934a6cf24d9f8dc018563abb/aws/corehandlers/handlers.go#L68-L108
// The customization is that instead of using the request's HTTP client directly to perform the HTTP
// request, the send handler uses the x/net/context/ctxhttp package to do this while respecting the
// context's deadline.
func InContext(ctx context.Context, req *request.Request) error {
	sendHandler := func(r *request.Request) {
		var reStatusCode = regexp.MustCompile(`^(\d{3})`)
		var err error
		r.HTTPResponse, err = ctxhttp.Do(ctx, r.Config.HTTPClient, r.HTTPRequest)
		if err != nil {
			// Prevent leaking if an HTTPResponse was returned. Clean up
			// the body.
			if r.HTTPResponse != nil {
				r.HTTPResponse.Body.Close()
			}
			// When the context expired, we don't want to retry the request.
			if err == ctx.Err() {
				r.Error = err
				r.Retryable = aws.Bool(false)
				return
			}
			// Capture the case where url.Error is returned for error processing
			// response. e.g. 301 without location header comes back as string
			// error and r.HTTPResponse is nil. Other url redirect errors will
			// comeback in a similar method.
			if e, ok := err.(*url.Error); ok && e.Err != nil {
				if s := reStatusCode.FindStringSubmatch(e.Err.Error()); s != nil {
					code, _ := strconv.ParseInt(s[1], 10, 64)
					r.HTTPResponse = &http.Response{
						StatusCode: int(code),
						Status:     http.StatusText(int(code)),
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					}
					return
				}
			}
			if r.HTTPResponse == nil {
				// Add a dummy request response object to ensure the HTTPResponse
				// value is consistent.
				r.HTTPResponse = &http.Response{
					StatusCode: int(0),
					Status:     http.StatusText(int(0)),
					Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
				}
			}
			// Catch all other request errors.
			r.Error = awserr.New("RequestError", "send request failed", err)
			r.Retryable = aws.Bool(true) // network errors are retryable
		}
	}

	req.Handlers.Send.Remove(corehandlers.SendHandler)
	req.Handlers.Send.PushBack(sendHandler)
	return req.Send()
}
