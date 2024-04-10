package main

import (
	"testing"
)

func verifyGot(t *testing.T, name string, args string, wanted any, got any) {
	if got != wanted {
		t.Errorf("%v failed, got %q wanted %q for %q", name, got, wanted, args)
	}
}

func Test_standardizeFileRef(t *testing.T) {
	got := standardizeFileRef("/dev/null")
	verifyGot(t, "standardizeFileRef", "/dev/null", "/dev/null", got)
	got = standardizeFileRef("null")
	verifyGot(t, "standardizeFileRef", "null", "/dev/null", got)
	got = standardizeFileRef("")
	verifyGot(t, "standardizeFileRef", "", "/dev/null", got)

	got = standardizeFileRef("/dev/stdout")
	verifyGot(t, "standardizeFileRef", "/dev/stdout", "/dev/stdout", got)
	got = standardizeFileRef("stdout")
	verifyGot(t, "standardizeFileRef", "stdout", "/dev/stdout", got)
	got = standardizeFileRef("-")
	verifyGot(t, "standardizeFileRef", "-", "/dev/stdout", got)

	got = standardizeFileRef("/dev/stderr")
	verifyGot(t, "standardizeFileRef", "/dev/stderr", "/dev/stderr", got)
	got = standardizeFileRef("stderr")
	verifyGot(t, "standardizeFileRef", "stderr", "/dev/stderr", got)

	got = standardizeFileRef("/boo")
	verifyGot(t, "standardizeFileRef", "/boo", "/boo", got)
}
