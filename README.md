# ctxaws - Perform AWS requests while respecting contexts

When using the [`net/context`](https://godoc.org/golang.org/x/net/context) package, one should 
respect context expiration when performing requests to backing services. Typically, a
`context.Context` is created for every incoming request and all calls to other services that happen
during this request have to be cancelled when the request's context expires.

The `ctxaws` package includes a helper function `ctxaws.InContext` that performs AWS API requests
while respecting a context's cancellation.

Support for the context pattern in the SDK directly [was discussed](https://github.com/aws/aws-sdk-go/issues/75)
but not implemented.  

## Usage

In Go projects using the AWS SDK, requests or most typically performed like this: 

```go
func countItems() (int, error) {
    out, err := client.Scan(&dynamodb.ScanInput{
        TableName: aws.String("my-table"),
    })
    if err != nil {
        return 0, err   
    }
    return len(out.Items), nil
}
```

DynamoDB is just used as an example here.

The **Problem** with such SDK operations is that they lead to HTTP requests that cannot be cancelled
and are retried for a (configurable) number of times until they succeed. In most cases, these
retries are desired behavior, however it cannot be controlled how long the whole operation takes.

In order to make the same API call while respecting a context's cancellation, one can use the
`ctxaws` package like this:

```go
func countItems(ctx context.Context) (int, error) {
    req, out := client.ScanRequest(&dynamodb.ScanInput{
        TableName: aws.String("my-table"),
    })
    if err := ctxaws.InContext(ctx, req); err != nil {
        return 0, err   
    }
    return len(out.Items), nil
}
```

By using the special `*Request` methods of the AWS service clients, we can delegate the processing 
of the request to `ctxhttp` which makes sure that the context cancellation is respected while 
performing the request.

## Development

There is a `cmd/main` package with an example program that can be used for testing. Please note that
it requires a DynamoDB table named `test-table` that needs to be created before running it. When 
run it uses the AWS credentials exported on the command line to connect to DynamoDB.

```bash
$ go run cmd/main/main.go
```