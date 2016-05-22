package ctxaws_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/seiffert/ctxaws"
)

func TestInContext_Success(t *testing.T) {
	// set up test-server that always responds successfully
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	cfg := aws.NewConfig().WithEndpoint(server.URL)
	client := dynamodb.New(session.New(cfg))
	req, _ := client.ScanRequest(&dynamodb.ScanInput{
		TableName: aws.String("test-table"),
	})

	// create context that is not cancelled before the server responds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// perform the request
	err := ctxaws.InContext(ctx, req)
	if err != nil {
		t.Fatalf("An error occurred: %s", err)
	}
}

func TestInContext_SlowServer(t *testing.T) {
	// set up test-server that is really slow
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	cfg := aws.NewConfig().WithEndpoint(server.URL)
	client := dynamodb.New(session.New(cfg))
	req, _ := client.ScanRequest(&dynamodb.ScanInput{
		TableName: aws.String("test-table"),
	})

	// create context that is cancelled before the server responds
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// perform the request
	err := ctxaws.InContext(ctx, req)
	if err == nil {
		t.Fatalf("No error occurred when context was cancelled")
	}
	if err != ctx.Err() {
		t.Fatalf("The error that occurred was not the context cancellation error: %s", err)
	}
}

func TestInContext_ServerError(t *testing.T) {
	// set up test-server that always returns an error after one second
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	cfg := aws.NewConfig().WithEndpoint(server.URL)
	client := dynamodb.New(session.New(cfg))
	req, _ := client.ScanRequest(&dynamodb.ScanInput{
		TableName: aws.String("test-table"),
	})

	// create context that is not cancelled before the server responds
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// perform the request
	err := ctxaws.InContext(ctx, req)
	if err == nil {
		t.Fatalf("No error occurred when context was cancelled")
	}
	if err != ctx.Err() {
		t.Fatalf("The error that occurred was not the context cancellation error: %s", err)
	}
}
