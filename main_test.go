package main

import (
	"testing"
)

func helper(t *testing.T, name string, args string, wanted any, got any) {
	if got != wanted {
		t.Errorf("%v failed, got %q wanted %q for %q", name, got, wanted, args)
	}
}

func Test_standardizeFileRef(t *testing.T) {
	got := standardizeFileRef("/dev/null")
	helper(t, "standardizeFileRef", "/dev/null", "/dev/null", got)
	got = standardizeFileRef("null")
	helper(t, "standardizeFileRef", "null", "/dev/null", got)
	got = standardizeFileRef("")
	helper(t, "standardizeFileRef", "", "/dev/null", got)

	got = standardizeFileRef("/dev/stdout")
	helper(t, "standardizeFileRef", "/dev/stdout", "/dev/stdout", got)
	got = standardizeFileRef("stdout")
	helper(t, "standardizeFileRef", "stdout", "/dev/stdout", got)
	got = standardizeFileRef("-")
	helper(t, "standardizeFileRef", "-", "/dev/stdout", got)

	got = standardizeFileRef("/dev/stderr")
	helper(t, "standardizeFileRef", "/dev/stderr", "/dev/stderr", got)
	got = standardizeFileRef("stderr")
	helper(t, "standardizeFileRef", "stderr", "/dev/stderr", got)

	got = standardizeFileRef("/boo")
	helper(t, "standardizeFileRef", "/boo", "/boo", got)
}
