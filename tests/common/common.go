package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

func AssertArraysEqual(t *testing.T, wanted []string, got []string) {
	if len(wanted) != len(got) {
		t.Error("Array lengths did not match, will not check")
	}
	for i := range wanted {
		AssertEqual(t, wanted[i], got[i])
	}
}
func AssertEqual(t *testing.T, wanted any, got any) {
	if got != wanted {
		t.Errorf("Got %q, but wanted %q", got, wanted)
	}
}

// helper functions
func VerifyGot(t *testing.T, wanted any, got any) bool {
	if got != wanted {
		t.Errorf("got %q wanted %q", got, wanted)
		return false
	}
	return true
}
func VerifyJson(t *testing.T, json map[string]interface{}, arg string) bool {
	if json[arg] == nil {
		err := fmt.Sprintf("%v was not present in json response", arg)
		t.Errorf(err)
		return false
	}
	return true
}

func ReadJson(file string) (res map[string]interface{}, raw string, err error) {
	jsonFile, err := os.Open(file)
	if err != nil {
		return nil, "", err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, "", err
	}

	raw = string(byteValue)
	json.Unmarshal(byteValue, &res)
	return res, raw, nil
}
