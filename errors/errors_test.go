package errors

import (
	"errors"
	"testing"
)

func Test_ErrorCollection(t *testing.T) {
	cc := new(CurlErrorCollection)
	if cc.Errors != nil {
		t.Error("cc.Errors should be nil")
	}

	cc.AppendError(-1, errors.New("testing"))
	if cc.Errors == nil || len(cc.Errors) != 1 {
		t.Error("cc.Errors should be 1 long")
	}
	if cc.Errors[0].ErrorString != "testing" {
		t.Error("cc.Errors[0] should be 'testing'")
	}
	if cc.Errors[0].Error() != "testing" {
		t.Error("cc.Errors[0] should be 'testing'")
	}
	if cc.Errors[0].ExitCode != -1 {
		t.Error("cc.Errors[0].ExitCode should be -1")
	}

	cc.AppendCurlError(NewCurlErrorFromString(-2, "testing2"))
	if cc.Errors == nil || len(cc.Errors) != 2 {
		t.Error("cc.Errors should be 2 long")
	}
	if cc.Errors[1].ErrorString != "testing2" {
		t.Error("cc.Errors[1] should be 'testing2'")
	}
	if cc.Errors[1].Error() != "testing2" {
		t.Error("cc.Errors[1] should be 'testing2'")
	}
	if cc.Errors[1].ExitCode != -2 {
		t.Error("cc.Errors[1].ExitCode should be -2")
	}

	cc.AppendCurlError(NewCurlErrorFromError(-3, errors.New("testing3")))
	if cc.Errors == nil || len(cc.Errors) != 3 {
		t.Error("cc.Errors should be 3 long")
	}
	if cc.Errors[2].ErrorString != "testing3" {
		t.Error("cc.Errors[2] should be 'testing3'")
	}
	if cc.Errors[2].Error() != "testing3" {
		t.Error("cc.Errors[2] should be 'testing3'")
	}
	if cc.Errors[2].ExitCode != -3 {
		t.Error("cc.Errors[2].ExitCode should be -3")
	}

	cc.AppendCurlError(NewCurlErrorFromStringAndError(-4, "testingfour", errors.New("testing4")))
	if cc.Errors == nil || len(cc.Errors) != 4 {
		t.Error("cc.Errors should be 2 long")
	}
	if cc.Errors[3].ErrorString != "testingfour: testing4" {
		t.Error("cc.Errors[3] should be 'testingfour: testing4'")
	}
	if cc.Errors[3].Error() != "testingfour: testing4" {
		t.Error("cc.Errors[3] should be 'testingfour: testing4'")
	}
	if cc.Errors[3].ExitCode != -4 {
		t.Error("cc.Errors[3].ExitCode should be -4")
	}

	cc.AppendCurlErrors(cc)
	if cc.Errors == nil || len(cc.Errors) != 8 {
		t.Error("cc.Errors should be 8 long")
	}
}
