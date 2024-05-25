package cli

import (
	"os"
	"path/filepath"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlcommontests "github.com/cdwiegand/go-curling/tests/common"
)

func Test_ParseConfigFileLine(t *testing.T) {
	curlcommontests.AssertArraysEqual(t, []string{}, ParseConfigLine(""))
	curlcommontests.AssertArraysEqual(t, []string{}, ParseConfigLine("# this is a comment"))
	curlcommontests.AssertArraysEqual(t, []string{"-v"}, ParseConfigLine("-v"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose"}, ParseConfigLine("--verbose"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "1"}, ParseConfigLine("--verbose 1"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose"}, ParseConfigLine("verbose"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "1"}, ParseConfigLine("verbose 1"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "1"}, ParseConfigLine("verbose= 1"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "1"}, ParseConfigLine("verbose =1"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "1"}, ParseConfigLine("verbose = 1"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "1"}, ParseConfigLine("verbose: 1"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "1"}, ParseConfigLine("verbose :1"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "1"}, ParseConfigLine("verbose : 1"))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose= \"one two three\""))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose =\"one two three\""))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose = \"one two three\""))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose: \"one two three\""))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose :\"one two three\""))
	curlcommontests.AssertArraysEqual(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose : \"one two three\""))
	curlcommontests.AssertArraysEqual(t, []string{"--url", "https://httpbin.org/post"}, ParseConfigLine("url = \"https://httpbin.org/post\""))
}

func Test_ParseArgsWithConfigFile(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "config.test")
	config := "-v"
	config = config + "\n--silent"
	config += "\nrequest POST"
	config += "\nurl = \"https://httpbin.org/post\""
	os.WriteFile(testFile, []byte(config), 0666)

	args := []string{"-i", "-K", testFile}
	ctx := new(curl.CurlContext)
	extras, err := ParseFlags(args, ctx)
	if err != nil {
		t.Error(err)
	}
	if len(extras) > 0 {
		t.Errorf("Got %d extras, shouldn't have any", len(extras))
	}
	curlcommontests.AssertEqual(t, true, ctx.IncludeHeadersInMainOutput)    // -i
	curlcommontests.AssertEqual(t, true, ctx.Verbose)                       // -v
	curlcommontests.AssertEqual(t, true, ctx.IsSilent)                      // --silent
	curlcommontests.AssertEqual(t, "POST", ctx.HttpVerb)                    // request POST
	curlcommontests.AssertEqual(t, 1, len(ctx.Urls))                        // url = "https://httpbin.org/post"
	curlcommontests.AssertEqual(t, "https://httpbin.org/post", ctx.Urls[0]) // url = "https://httpbin.org/post"
}
