package cli

import (
	"os"
	"path/filepath"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	"github.com/stretchr/testify/assert"
)

func Test_ParseConfigFileLine(t *testing.T) {
	assert.EqualValues(t, []string{}, ParseConfigLine(""))
	assert.EqualValues(t, []string{}, ParseConfigLine("# this is a comment"))
	assert.EqualValues(t, []string{"-v"}, ParseConfigLine("-v"))
	assert.EqualValues(t, []string{"--verbose"}, ParseConfigLine("--verbose"))
	assert.EqualValues(t, []string{"--verbose", "1"}, ParseConfigLine("--verbose 1"))
	assert.EqualValues(t, []string{"--verbose"}, ParseConfigLine("verbose"))
	assert.EqualValues(t, []string{"--verbose", "1"}, ParseConfigLine("verbose 1"))
	assert.EqualValues(t, []string{"--verbose", "1"}, ParseConfigLine("verbose= 1"))
	assert.EqualValues(t, []string{"--verbose", "1"}, ParseConfigLine("verbose =1"))
	assert.EqualValues(t, []string{"--verbose", "1"}, ParseConfigLine("verbose = 1"))
	assert.EqualValues(t, []string{"--verbose", "1"}, ParseConfigLine("verbose: 1"))
	assert.EqualValues(t, []string{"--verbose", "1"}, ParseConfigLine("verbose :1"))
	assert.EqualValues(t, []string{"--verbose", "1"}, ParseConfigLine("verbose : 1"))
	assert.EqualValues(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose= \"one two three\""))
	assert.EqualValues(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose =\"one two three\""))
	assert.EqualValues(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose = \"one two three\""))
	assert.EqualValues(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose: \"one two three\""))
	assert.EqualValues(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose :\"one two three\""))
	assert.EqualValues(t, []string{"--verbose", "one two three"}, ParseConfigLine("verbose : \"one two three\""))
	assert.EqualValues(t, []string{"--url", "https://httpbin.org/post"}, ParseConfigLine("url = \"https://httpbin.org/post\""))
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
	assert.Equal(t, true, ctx.IncludeHeadersInMainOutput)    // -i
	assert.Equal(t, true, ctx.Verbose)                       // -v
	assert.Equal(t, true, ctx.IsSilent)                      // --silent
	assert.Equal(t, "POST", ctx.HttpVerb)                    // request POST
	assert.Equal(t, 1, len(ctx.Urls))                        // url = "https://httpbin.org/post"
	assert.Equal(t, "https://httpbin.org/post", ctx.Urls[0]) // url = "https://httpbin.org/post"
}
