package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// helper functions
func VerifyGot(t *testing.T, wanted any, got any) {
	if got != wanted {
		t.Errorf("got %q wanted %q", got, wanted)
	}
}
func VerifyJson(t *testing.T, json map[string]interface{}, arg string) {
	if json[arg] == nil {
		err := fmt.Sprintf("%v was not present in json response", arg)
		t.Errorf(err)
		panic(err)
	}
}

func ReadJson(file string) (res map[string]interface{}) {
	jsonFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}
	json.Unmarshal([]byte(byteValue), &res)
	return
}

func BuildFileList(count int, outputDir string, ext string) (files []string) {
	files = []string{}
	for i := 0; i < count; i++ {
		files = append(files, filepath.Join(outputDir, fmt.Sprintf("%d.%s", i, ext)))
	}
	return
}
