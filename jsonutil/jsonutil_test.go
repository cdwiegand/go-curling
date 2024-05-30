package jsonutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.True(t, Equal(res, res2, func(path string) bool {
		return path == "Ignore"
	}))
}

func Test_Delete(t *testing.T) {
	jsonStr := "{\"Testing\":1,\"Hello\":\"World\"}"
	byteValue := []byte(jsonStr)

	var res map[string]interface{}
	json.Unmarshal(byteValue, &res)

	Remove(res, "Testing")
	assert.Nil(t, res["Testing"])
	assert.NotNil(t, res["Hello"])

	Remove(res, "Hello")
	assert.Nil(t, res["Hello"])
}
