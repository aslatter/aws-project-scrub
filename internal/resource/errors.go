package resource

import (
	"errors"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
)

func IsErrNotFound(err error) bool {
	var httpResponseErr *awshttp.ResponseError
	if errors.As(err, &httpResponseErr) {
		return httpResponseErr.Response.StatusCode == 404
	}
	return false
}
