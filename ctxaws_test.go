package ctxaws_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"golang.org/x/net/context"

	"github.com/seiffert/ctxaws"
)

func TestInContext_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := scanTableWithServer(ctx, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err != nil {
		t.Fatalf("An error occurred: %s", err)
	}
}

func TestInContext_SlowServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := scanTableWithServer(ctx, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	})
	if err == nil {
		t.Fatalf("No error occurred when context was cancelled")
	}
	if err != ctx.Err() {
		t.Fatalf("The error that occurred was not the context cancellation error: %s", err)
	}
}

func scanTableWithServer(ctx context.Context, handler func(http.ResponseWriter, *http.Request)) error {
	server := httptest.NewServer(http.HandlerFunc(handler))
	cfg := aws.NewConfig().
		WithEndpoint(server.URL).
		WithCredentials(credentials.NewStaticCredentials("AKID", "SECRET", ""))
	client := dynamodb.New(session.New(cfg))
	req, _ := client.ScanRequest(&dynamodb.ScanInput{})

	req.Handlers.Validate.Clear()

	return ctxaws.InContext(ctx, req)
}
