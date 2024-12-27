package resource

import (
	"errors"
)

func IsErrNotFound(err error) bool {
	var httpResponseErr interface{ HTTPStatusCode() int }
	if errors.As(err, &httpResponseErr) && httpResponseErr.HTTPStatusCode() == 404 {
		return true
	}

	// EC2 resources seem to use status-code '400' for not-found errors, so we check for
	// error-codes explicitly. This doesn't make sense for a general 'not found' check,
	// but in the context of resource-deletions it is what we want.
	//
	// So far we've seen this with:
	//  + Security Groups
	//  + Volumes
	//
	// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html#CommonErrors
	var apiError interface{ ErrorCode() string }
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "InvalidGroup.NotFound", "InvalidVolume.NotFound":
			return true
		}
	}

	return false
}
