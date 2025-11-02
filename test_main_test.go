package main

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
)

func Test_reportError(t *testing.T) {
	ctx := new(curl.CurlContext)
	got := reportError(nil, ctx)
	wanted := ""
	if got != wanted {
		t.Errorf("Wanted '%q' but got '%q'", wanted, got)
	}

	testError := curlerrors.NewCurlErrorFromString(-1, "Testing 1 2 3 4")
	got = reportError(testError, ctx)
	wanted = "Error: Testing 1 2 3 4.\n"
	if got != wanted {
		t.Errorf("Wanted '%q' but got '%q'", wanted, got)
	}
}
