package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newVarCtx(t *testing.T, specs ...string) *CurlContext {
	t.Helper()
	ctx := new(CurlContext)
	for _, s := range specs {
		require.NoError(t, ctx.SetVariable(s), s)
	}
	return ctx
}

func expandOK(t *testing.T, ctx *CurlContext, in string) string {
	t.Helper()
	out, err := ctx.ExpandVariables(in)
	require.NoError(t, err, in)
	return out
}

// --- defining variables ---

func TestSetVariable_LiteralContent(t *testing.T) {
	for _, tc := range []struct{ spec, name, want string }{
		{"a=1", "a", "1"},
		{"name=hello world", "name", "hello world"},
		{"empty=", "empty", ""},
		{"eq=a=b=c", "eq", "a=b=c"},             // only the first = separates the name
		{"at=x@y", "at", "x@y"},                 // = wins when it comes first
		{"Mixed_Case_9=v", "Mixed_Case_9", "v"}, // letters, digits, underscore
		{"_=underscore", "_", "underscore"},     // a lone underscore is a valid name
		{"9lives=cat", "9lives", "cat"},         // names may start with a digit
		{"braces={{x}}", "braces", "{{x}}"},     // content is stored literally, never expanded
		{"spaces=  padded  ", "spaces", "  padded  "},
		{"utf8=héllo ✓", "utf8", "héllo ✓"},
	} {
		t.Run(tc.spec, func(t *testing.T) {
			ctx := newVarCtx(t, tc.spec)
			require.Equal(t, tc.want, ctx.Variables[tc.name])
			require.Empty(t, ctx.Warnings)
		})
	}
}

func TestSetVariable_LastDefinitionWins(t *testing.T) {
	ctx := newVarCtx(t, "a=1", "a=2", "b=x", "a=3")
	require.Equal(t, "3", ctx.Variables["a"])
	require.Equal(t, "x", ctx.Variables["b"])
}

func TestSetVariable_NamesAreCaseSensitive(t *testing.T) {
	ctx := newVarCtx(t, "Key=upper", "key=lower")
	require.Equal(t, "upper", ctx.Variables["Key"])
	require.Equal(t, "lower", ctx.Variables["key"])
}

func TestSetVariable_FromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "content.txt")
	require.NoError(t, os.WriteFile(file, []byte("line1\nline2\n"), 0600))
	ctx := newVarCtx(t, "f@"+file)
	require.Equal(t, "line1\nline2\n", ctx.Variables["f"]) // file content is kept byte for byte

	bin := filepath.Join(dir, "bin.dat")
	require.NoError(t, os.WriteFile(bin, []byte{0, 1, 2, 0xff}, 0600))
	ctx = newVarCtx(t, "b@"+bin)
	require.Equal(t, "\x00\x01\x02\xff", ctx.Variables["b"]) // NUL bytes may be stored

	empty := filepath.Join(dir, "empty")
	require.NoError(t, os.WriteFile(empty, nil, 0600))
	ctx = newVarCtx(t, "e@"+empty)
	v, ok := ctx.Variables["e"]
	require.True(t, ok)
	require.Equal(t, "", v)
}

func TestSetVariable_FileErrors(t *testing.T) {
	ctx := new(CurlContext)
	err := ctx.SetVariable("f@" + filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to read variable file")
	require.Error(t, ctx.SetVariable("f@")) // no file name
}

func TestSetVariable_FromEnvironment(t *testing.T) {
	t.Setenv("GOCURL_VAR_SET", "from-env")
	t.Setenv("GOCURL_VAR_EMPTY", "")
	os.Unsetenv("GOCURL_VAR_UNSET")
	dir := t.TempDir()
	defaultFile := filepath.Join(dir, "default.txt")
	require.NoError(t, os.WriteFile(defaultFile, []byte("file-default"), 0600))

	for _, tc := range []struct{ spec, name, want string }{
		{"%GOCURL_VAR_SET", "GOCURL_VAR_SET", "from-env"},
		{"%GOCURL_VAR_SET=ignored-default", "GOCURL_VAR_SET", "from-env"},        // env wins over default
		{"%GOCURL_VAR_SET@" + defaultFile, "GOCURL_VAR_SET", "from-env"},         // env wins over file default
		{"%GOCURL_VAR_EMPTY", "GOCURL_VAR_EMPTY", ""},                            // set-but-empty counts as set
		{"%GOCURL_VAR_EMPTY=dflt", "GOCURL_VAR_EMPTY", ""},                       // ...so the default is not used
		{"%GOCURL_VAR_UNSET=fallback", "GOCURL_VAR_UNSET", "fallback"},           // default used when unset
		{"%GOCURL_VAR_UNSET=", "GOCURL_VAR_UNSET", ""},                           // empty default
		{"%GOCURL_VAR_UNSET@" + defaultFile, "GOCURL_VAR_UNSET", "file-default"}, // default from file
	} {
		t.Run(tc.spec, func(t *testing.T) {
			ctx := newVarCtx(t, tc.spec)
			v, ok := ctx.Variables[tc.name]
			require.True(t, ok)
			require.Equal(t, tc.want, v)
		})
	}
}

func TestSetVariable_UnsetEnvironmentWithoutDefaultIsAnError(t *testing.T) {
	os.Unsetenv("GOCURL_VAR_UNSET")
	ctx := new(CurlContext)
	err := ctx.SetVariable("%GOCURL_VAR_UNSET")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not set")
	require.NotContains(t, ctx.Variables, "GOCURL_VAR_UNSET")
}

func TestSetVariable_BadSyntaxWarnsAndSkips(t *testing.T) {
	long := strings.Repeat("x", MaxVariableNameLength+1)
	for _, spec := range []string{
		"novalue",       // neither = nor @
		"bad-name=1",    // '-' ends the name, then neither = nor @ follows
		"bad name=1",    // space is not a name character
		"=nameless",     // empty name
		"",              // nothing at all
		"%",             // empty environment name
		"%=x",           // empty environment name with default
		long + "=value", // name too long
		"a.b=1",
		"a:b=1",
	} {
		t.Run(spec, func(t *testing.T) {
			ctx := new(CurlContext)
			require.NoError(t, ctx.SetVariable(spec))
			require.Empty(t, ctx.Variables)
			require.Len(t, ctx.Warnings, 1)
			require.Contains(t, ctx.Warnings[0], "skipping")
		})
	}
}

func TestSetVariable_MaxNameLengthAccepted(t *testing.T) {
	name := strings.Repeat("n", MaxVariableNameLength)
	ctx := newVarCtx(t, name+"=ok")
	require.Equal(t, "ok", ctx.Variables[name])
	require.Equal(t, "ok", expandOK(t, ctx, "{{"+name+"}}"))
}

func TestSetVariable_ByteRanges(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "digits.txt")
	require.NoError(t, os.WriteFile(file, []byte("0123456789"), 0600))
	t.Setenv("GOCURL_VAR_RANGE", "abcdefgh")

	for _, tc := range []struct{ spec, name, want string }{
		{"r[0-0]=abcdef", "r", "a"},
		{"r[1-3]=abcdef", "r", "bcd"}, // inclusive on both ends
		{"r[2-]=abcdef", "r", "cdef"}, // open-ended
		{"r[0-]=abcdef", "r", "abcdef"},
		{"r[4-100]=abcdef", "r", "ef"}, // end past content is clamped
		{"r[6-]=abcdef", "r", ""},      // start at length gives nothing
		{"r[50-60]=abcdef", "r", ""},   // start past content gives nothing
		{"r[3-5]@" + file, "r", "345"}, // from a file
		{"r[8-]@" + file, "r", "89"},
		{"%GOCURL_VAR_RANGE[2-4]", "GOCURL_VAR_RANGE", "cde"}, // from the environment
	} {
		t.Run(tc.spec, func(t *testing.T) {
			ctx := newVarCtx(t, tc.spec)
			require.Equal(t, tc.want, ctx.Variables[tc.name])
		})
	}
}

func TestSetVariable_BadByteRanges(t *testing.T) {
	for _, spec := range []string{
		"r[3-1]=abcdef", // end before start
		"r[-3]=abcdef",  // missing start
		"r[a-b]=abcdef", // not numbers
		"r[1]=abcdef",   // no dash
		"r[1-2=abcdef",  // unterminated
		"r[-1-2]=abc",   // negative
	} {
		t.Run(spec, func(t *testing.T) {
			ctx := new(CurlContext)
			require.Error(t, ctx.SetVariable(spec))
			require.Empty(t, ctx.Variables)
		})
	}
}

// --- expanding variables ---

func TestExpand_Basics(t *testing.T) {
	ctx := newVarCtx(t, "a=1", "name=world", "USER_ID=12345", "empty=")
	for _, tc := range []struct{ in, want string }{
		{"", ""},
		{"no references here", "no references here"},
		{"{{a}}", "1"},
		{"hello {{name}}!", "hello world!"},
		{"https://example.com{{USER_ID}}", "https://example.com12345"},
		{"https://example.com/users/{{USER_ID}}/profile", "https://example.com/users/12345/profile"},
		{"{{a}}{{a}}{{a}}", "111"},                      // adjacent references
		{"{{a}}-{{name}}-{{USER_ID}}", "1-world-12345"}, // several different references
		{"[{{empty}}]", "[]"},                           // defined but empty
		{"[{{undefined}}]", "[]"},                       // undefined expands to nothing
		{"{{A}}", ""},                                   // names are case sensitive
		{"{a}", "{a}"},                                  // single braces are left alone
		{"{{a}}}}", "1}}"},                              // stray closing braces are literal
		{"}}{{a}}", "}}1"},
		{"{{{a}}", "{{{a}}"}, // "{a" is not a valid name, so kept verbatim
	} {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, expandOK(t, ctx, tc.in))
		})
	}
}

func TestExpand_UndefinedVariableWithNoVariablesAtAll(t *testing.T) {
	ctx := new(CurlContext) // Variables map is nil
	require.Equal(t, "x--y", expandOK(t, ctx, "x-{{nothing}}-y"))
}

func TestExpand_ValueIsNotReExpanded(t *testing.T) {
	ctx := newVarCtx(t, "inner=secret", "outer={{inner}}")
	require.Equal(t, "{{inner}}", expandOK(t, ctx, "{{outer}}"))
}

func TestExpand_BackslashEscape(t *testing.T) {
	ctx := newVarCtx(t, "a=1")
	require.Equal(t, "{{a}}", expandOK(t, ctx, `\{{a}}`))
	require.Equal(t, "x{{a}}y", expandOK(t, ctx, `x\{{a}}y`))
	require.Equal(t, "q1{{a}}", expandOK(t, ctx, `q{{a}}\{{a}}`))
	require.Equal(t, "{{a}}1", expandOK(t, ctx, `\{{a}}{{a}}`))
	require.Equal(t, `\{{a}}`, expandOK(t, ctx, `\\{{a}}`)) // only the backslash right before {{ is consumed; the reference stays literal
	require.Equal(t, `a\b1`, expandOK(t, ctx, `a\b{{a}}`))
}

func TestExpand_MalformedReferencesAreKeptVerbatim(t *testing.T) {
	ctx := newVarCtx(t, "a=1")
	for _, tc := range []struct{ in, want string }{
		{"{{a", "{{a"}, // no closing braces
		{"pre{{a", "pre{{a"},
		{"{{a}}x{{a", "1x{{a"}, // only the unterminated tail is kept
		{"{{}}", "{{}}"},       // empty name
		{"{{a b}}", "{{a b}}"}, // invalid characters
		{"{{a-b}}", "{{a-b}}"},
		{"{{ a }}", "{{ a }}"},
		{"{{:trim}}", "{{:trim}}"}, // a function without a name
		{"{{a b}}|{{}}|{{a-b}}|{{a", "{{a b}}|{{}}|{{a-b}}|{{a"},
		{"{{" + strings.Repeat("z", MaxVariableNameLength+1) + "}}", "{{" + strings.Repeat("z", MaxVariableNameLength+1) + "}}"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			ctx.Warnings = nil
			require.Equal(t, tc.want, expandOK(t, ctx, tc.in))
			require.NotEmpty(t, ctx.Warnings, "a malformed reference should produce a warning")
		})
	}
}

func TestExpand_WellFormedReferencesProduceNoWarnings(t *testing.T) {
	ctx := newVarCtx(t, "a=1")
	expandOK(t, ctx, "{{a}} {{missing}} {{a:trim:url}} \\{{a}}")
	require.Empty(t, ctx.Warnings)
}

func TestExpand_Functions(t *testing.T) {
	ctx := newVarCtx(t,
		"plain=hello",
		"padded= \t\n hi there \r\n\v\f",
		"special=a b&c=d/e?f#g%h+i~j-k.l_m",
		"quote=say \"hi\" \\ back",
		"ctrl=tab\there\nnew\rret\bbs\fff\x01\x1f",
		"unicode=héllo ✓ /<>&",
		"spaced=  needs encoding & trim  ",
		"b64src=hello world",
		"b64enc=aGVsbG8gd29ybGQ=",
		"bad64=not*base64!",
		"empty=",
	)
	for _, tc := range []struct{ in, want string }{
		// trim
		{"[{{padded:trim}}]", "[hi there]"},
		{"[{{plain:trim}}]", "[hello]"},
		{"[{{empty:trim}}]", "[]"},
		{"[{{undefined:trim}}]", "[]"},
		// url: everything but ALPHA DIGIT - . _ ~ is percent-encoded, upper-case hex
		{"{{plain:url}}", "hello"},
		{"{{special:url}}", "a%20b%26c%3Dd%2Fe%3Ff%23g%25h%2Bi~j-k.l_m"},
		{"{{unicode:url}}", "h%C3%A9llo%20%E2%9C%93%20%2F%3C%3E%26"},
		// json: only quotes, backslashes and control characters are escaped
		{"{{quote:json}}", `say \"hi\" \\ back`},
		{"{{ctrl:json}}", `tab\there\nnew\rret\bbs\fff\u0001\u001f`},
		{"{{unicode:json}}", "héllo ✓ /<>&"},
		// b64
		{"{{b64src:b64}}", "aGVsbG8gd29ybGQ="},
		{"{{empty:b64}}", ""},
		// 64dec
		{"{{b64enc:64dec}}", "hello world"},
		{"{{bad64:64dec}}", "[64dec-fail]"},
		// chains are applied left to right
		{"{{spaced:trim:url}}", "needs%20encoding%20%26%20trim"},
		{"{{spaced:url:trim}}", "%20%20needs%20encoding%20%26%20trim%20%20"},
		{"{{b64src:b64:64dec}}", "hello world"},
		{"{{b64src:b64:url}}", "aGVsbG8gd29ybGQ%3D"},
		{"{{quote:json:b64}}", "c2F5IFwiaGlcIiBcXCBiYWNr"},
		{"{{padded:trim:trim:trim}}", "hi there"},
		// mixed with literal text and other references
		{"https://example.com/search?q={{special:url}}&u={{plain}}", "https://example.com/search?q=a%20b%26c%3Dd%2Fe%3Ff%23g%25h%2Bi~j-k.l_m&u=hello"},
		{`{"msg":"{{quote:json}}"}`, `{"msg":"say \"hi\" \\ back"}`},
	} {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, expandOK(t, ctx, tc.in))
		})
	}
}

func TestExpand_UnknownFunctionsAreErrors(t *testing.T) {
	ctx := newVarCtx(t, "a=x")
	for _, in := range []string{
		"{{a:bogus}}",
		"{{a:}}",         // empty function name
		"{{a::trim}}",    // empty function in a chain
		"{{a:trim:}}",    // trailing empty function
		"{{a:TRIM}}",     // function names are case sensitive
		"{{a:url:nope}}", // a later function in the chain
		"{{a: trim}}",
		"{{missing:bogus}}", // checked even when the variable is undefined
	} {
		t.Run(in, func(t *testing.T) {
			_, err := ctx.ExpandVariables(in)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unknown variable function")
		})
	}
}

func TestExpand_NullBytes(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nul.bin")
	require.NoError(t, os.WriteFile(file, []byte("a\x00b"), 0600))
	ctx := newVarCtx(t, "n@"+file, "enc=YQBi") // "YQBi" is base64 for "a\x00b"

	for _, in := range []string{"{{n}}", "{{n:trim}}", "x{{n}}y", "{{enc:64dec}}"} {
		t.Run("error "+in, func(t *testing.T) {
			_, err := ctx.ExpandVariables(in)
			require.Error(t, err)
			require.Contains(t, err.Error(), "null byte")
		})
	}
	for _, tc := range []struct{ in, want string }{
		{"{{n:b64}}", "YQBi"},
		{"{{n:url}}", "a%00b"},
		{"{{n:json}}", `a\u0000b`},
		{"{{enc:64dec:url}}", "a%00b"},
	} {
		t.Run("encoded "+tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, expandOK(t, ctx, tc.in))
		})
	}
}

func TestExpand_LiteralNulInInputIsNotAnError(t *testing.T) {
	// only variable values are checked; text around references passes through
	ctx := newVarCtx(t, "a=1")
	require.Equal(t, "\x001", expandOK(t, ctx, "\x00{{a}}"))
}

// --- encoders ---

func TestUrlEscapeVariable_AllBytes(t *testing.T) {
	for c := 0; c < 256; c++ {
		in := string([]byte{byte(c)})
		got := urlEscapeVariable(in)
		unreserved := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~'
		if unreserved {
			require.Equal(t, in, got)
		} else {
			require.Len(t, got, 3)
			require.Equal(t, byte('%'), got[0])
			require.Equal(t, strings.ToUpper(got), got)
		}
	}
}

func TestJsonQuoteVariable_AllControlBytes(t *testing.T) {
	named := map[byte]string{'\b': `\b`, '\f': `\f`, '\n': `\n`, '\r': `\r`, '\t': `\t`}
	for c := 0; c < 0x20; c++ {
		got := jsonQuoteVariable(string([]byte{byte(c)}))
		if want, ok := named[byte(c)]; ok {
			require.Equal(t, want, got)
		} else {
			require.Len(t, got, 6)
			require.True(t, strings.HasPrefix(got, `\u00`), got)
		}
	}
	for c := 0x20; c < 0x80; c++ {
		got := jsonQuoteVariable(string([]byte{byte(c)}))
		switch c {
		case '"':
			require.Equal(t, `\"`, got)
		case '\\':
			require.Equal(t, `\\`, got)
		default:
			require.Equal(t, string([]byte{byte(c)}), got)
		}
	}
}
