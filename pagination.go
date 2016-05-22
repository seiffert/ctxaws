package ctxaws

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"golang.org/x/net/context"
)

func PaginateInContext(ctx context.Context, req *request.Request, handlePage func(interface{}, bool) bool) error {
	if err := InContext(ctx, req); err != nil {
		return err
	}
	return req.EachPage(handlePage)
}
