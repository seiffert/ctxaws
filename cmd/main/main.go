package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/seiffert/ctxaws"
	"golang.org/x/net/context"
)

const (
	// TestContextTimeout is the timeout used for every request of the test.
	TestContextTimeout = 1 * time.Second

	// TestDynamoDBTable is the name of the DynamoDB table that is used for testing.
	TestDynamoDBTable = "test-table"

	// TestRequestTimeout is the timeout for all HTTP requests to DynamoDB.
	TestRequestTimeout = 300 * time.Millisecond
)

// This test performs many read and write operations on the table in order to reach the state in
// which the consumed capacity units exceed the provisioned ones and requests are being throttled.
// In order to succeed, requests should never be retried or continue running after the context's
// deadline has exceeded.
func main() {
	client := dynamodb.New(session.New(aws.NewConfig().WithHTTPClient(&http.Client{
		Timeout: TestRequestTimeout,
	})))
	// Writer
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), TestContextTimeout)
			defer cancel()

			if err := addItem(ctx, client); err != nil {
				log.Println("Error adding item:", err)
				continue
			}

			log.Println("Added 1 item")
		}
	}()
	// Reader
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), TestContextTimeout)
			defer cancel()

			count, err := countItems(ctx, client)
			if err != nil {
				log.Println("Error counting items:", err)
				continue
			}

			log.Println("Counted", count, "items")
		}
	}()
	// Keep the program running until interrupted
	for {
		select {}
	}
}

func addItem(ctx context.Context, client *dynamodb.DynamoDB) error {
	item, err := dynamodbattribute.MarshalMap(struct {
		Time time.Time
		Test string `dynamodbav:"test"`
	}{Time: time.Now(), Test: fmt.Sprintf("test-%d", time.Now().Unix())})
	if err != nil {
		return err
	}

	req, _ := client.PutItemRequest(&dynamodb.PutItemInput{
		TableName: aws.String(TestDynamoDBTable),
		Item:      item,
	})

	if err := ctxaws.InContext(ctx, req); err != nil {
		return err
	}
	return nil
}

func countItems(ctx context.Context, client *dynamodb.DynamoDB) (int, error) {
	count := 0
	req, _ := client.ScanRequest(&dynamodb.ScanInput{
		TableName: aws.String(TestDynamoDBTable),
	})
	if err := ctxaws.InContext(ctx, req); err != nil {
		return 0, err
	}
	err := req.EachPage(func(out interface{}, last bool) bool {
		count += len(out.(*dynamodb.ScanOutput).Items)
		return !last
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
