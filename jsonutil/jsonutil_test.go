package jsonutil

import (
	"encoding/json"
	"testing"
)

func Test_Equals(t *testing.T) {
	jsonStr := "{\"Testing\":1,\"Hello\":\"World\"}"
	byteValue := []byte(jsonStr)
	jsonStr2 := "{\"Testing\":1, \"Ignore\":\"Me\", \"Hello\":\"World\"}"
	byteValue2 := []byte(jsonStr2)

	var res map[string]interface{}
	var res2 map[string]interface{}

	json.Unmarshal(byteValue, &res)
	json.Unmarshal(byteValue2, &res2)
	if !Equal(res, res2, func(path string) bool {
		if path == "Ignore" {
			return true
		}
		return false
	}) {
		t.Fail()
	}
}
