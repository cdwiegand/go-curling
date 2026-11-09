package context

// Variables, like curl's --variable / --expand-* feature (curl 8.3.0+).
//
//	--variable name=content      set a variable from a literal value
//	--variable name@file         set a variable from a file's contents (@- reads stdin)
//	--variable %NAME             import environment variable NAME (error if unset)
//	--variable %NAME=default     import NAME, or use the default if unset
//	--variable %NAME@file        import NAME, or read the default from a file if unset
//	--variable 'name[N-M]=...'   keep only bytes N through M (inclusive) of the content; [N-] keeps N onward
//
// Any option that takes a value can be prefixed with --expand- (for example
// --expand-url) so that {{name}} in its value is replaced by the variable's
// contents. An undefined variable expands to nothing. Functions can be chained
// after the name: {{name:trim:url}}. Supported functions are trim, json, url,
// b64 and 64dec. A backslash before {{ keeps it literally: \{{.

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MaxVariableNameLength mirrors curl's limit on variable names.
const MaxVariableNameLength = 127

// SetVariable parses one --variable argument and stores the result. Like
// curl, malformed syntax is reported as a warning (see Warnings) and the
// argument is skipped; an unset environment variable without a default, a
// bad byte range or an unreadable file is an error.
func (ctx *CurlContext) SetVariable(spec string) error {
	if ctx.Variables == nil {
		ctx.Variables = map[string]string{}
	}
	fromEnv := strings.HasPrefix(spec, "%")
	rest := spec
	if fromEnv {
		rest = spec[1:]
	}

	nameEnd := 0
	for nameEnd < len(rest) && isVariableNameChar(rest[nameEnd]) {
		nameEnd++
	}
	name := rest[:nameEnd]
	if name == "" || len(name) > MaxVariableNameLength {
		ctx.addWarning(fmt.Sprintf("Bad variable name length (%d), skipping: %s", len(name), spec))
		return nil
	}
	rest = rest[nameEnd:]

	// optional byte range: [N-M] or [N-]
	hasRange := false
	var rangeStart, rangeEnd int64
	if strings.HasPrefix(rest, "[") {
		closeIdx := strings.Index(rest, "]")
		if closeIdx < 0 {
			return fmt.Errorf("bad byte range in --variable %q", spec)
		}
		var err error
		rangeStart, rangeEnd, err = parseVariableRange(rest[1:closeIdx])
		if err != nil {
			return fmt.Errorf("bad byte range in --variable %q: %w", spec, err)
		}
		hasRange = true
		rest = rest[closeIdx+1:]
	}

	var content string
	switch {
	case fromEnv && envIsSet(name):
		content = os.Getenv(name)
	case rest == "" && fromEnv:
		return fmt.Errorf("variable %q is not set in the environment", name)
	case strings.HasPrefix(rest, "="):
		content = rest[1:]
	case strings.HasPrefix(rest, "@"):
		file := rest[1:]
		if file == "" {
			return fmt.Errorf("missing file name in --variable %q", spec)
		}
		data, err := ctx.readInputFile(file)
		if err != nil {
			return fmt.Errorf("unable to read variable file %q: %w", file, err)
		}
		content = string(data)
	default:
		ctx.addWarning("Bad --variable syntax, skipping: " + spec)
		return nil
	}

	if hasRange {
		content = applyVariableRange(content, rangeStart, rangeEnd)
	}
	ctx.Variables[name] = content
	return nil
}

func (ctx *CurlContext) addWarning(msg string) {
	ctx.Warnings = append(ctx.Warnings, msg)
}

func envIsSet(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}

func parseVariableRange(r string) (start int64, end int64, err error) {
	parts := strings.SplitN(r, "-", 2)
	if len(parts) != 2 || parts[0] == "" {
		return 0, 0, fmt.Errorf("want [N-M] or [N-]")
	}
	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return 0, 0, fmt.Errorf("invalid start offset %q", parts[0])
	}
	if parts[1] == "" {
		return start, -1, nil
	}
	end, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil || end < 0 {
		return 0, 0, fmt.Errorf("invalid end offset %q", parts[1])
	}
	if end < start {
		return 0, 0, fmt.Errorf("end offset %d is before start offset %d", end, start)
	}
	return start, end, nil
}

// applyVariableRange keeps bytes start..end inclusive; end < 0 means to the end.
func applyVariableRange(content string, start int64, end int64) string {
	n := int64(len(content))
	if start >= n {
		return ""
	}
	if end < 0 || end >= n {
		return content[start:]
	}
	return content[start : end+1]
}

func isVariableNameChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isValidVariableName(name string) bool {
	if name == "" || len(name) > MaxVariableNameLength {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !isVariableNameChar(name[i]) {
			return false
		}
	}
	return true
}

// ExpandVariables replaces {{name}} and {{name:func:...}} references in input,
// following curl's rules: undefined names expand to nothing, malformed
// references and unterminated {{ are kept verbatim, \{{ yields a literal {{,
// unknown functions are an error, and so is a value that still contains a NUL
// byte after its functions are applied.
func (ctx *CurlContext) ExpandVariables(input string) (string, error) {
	var out strings.Builder
	line := input
	for {
		idx := strings.Index(line, "{{")
		if idx < 0 {
			out.WriteString(line)
			break
		}
		if idx > 0 && line[idx-1] == '\\' {
			out.WriteString(line[:idx-1])
			out.WriteString("{{")
			line = line[idx+2:]
			continue
		}
		inner := line[idx+2:]
		closeIdx := strings.Index(inner, "}}")
		if closeIdx < 0 {
			ctx.addWarning(fmt.Sprintf("missing close '}}' in '%s'", input))
			out.WriteString(line) // no closing braces: keep the rest as-is
			break
		}
		out.WriteString(line[:idx])
		ref := inner[:closeIdx]
		line = inner[closeIdx+2:]

		name, funcs, hasFuncs := strings.Cut(ref, ":")
		if !isValidVariableName(name) {
			ctx.addWarning("bad variable name: " + name)
			out.WriteString("{{" + ref + "}}") // not a variable reference
			continue
		}
		value := ctx.Variables[name]
		if hasFuncs {
			var err error
			value, err = applyVariableFunctions(name, value, funcs)
			if err != nil {
				return "", err
			}
		}
		if strings.IndexByte(value, 0) >= 0 {
			return "", fmt.Errorf("variable %q contains a null byte; use a function such as :b64, :url or :json to expand it", name)
		}
		out.WriteString(value)
	}
	return out.String(), nil
}

func applyVariableFunctions(name string, value string, funcs string) (string, error) {
	for _, fn := range strings.Split(funcs, ":") {
		switch fn {
		case "trim":
			value = strings.Trim(value, " \t\n\v\f\r")
		case "json":
			value = jsonQuoteVariable(value)
		case "url":
			value = urlEscapeVariable(value)
		case "b64":
			value = base64.StdEncoding.EncodeToString([]byte(value))
		case "64dec":
			decoded, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				value = "[64dec-fail]"
			} else {
				value = string(decoded)
			}
		default:
			return "", fmt.Errorf("unknown variable function %q in {{%s:%s}}", fn, name, funcs)
		}
	}
	return value, nil
}

// jsonQuoteVariable escapes content for use inside a JSON string (without the
// surrounding quotes), matching curl: only quotes, backslashes and control
// characters are escaped; other bytes pass through unchanged.
func jsonQuoteVariable(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

// urlEscapeVariable percent-encodes every byte except ALPHA, DIGIT, '-', '.',
// '_' and '~', as curl_easy_escape does.
func urlEscapeVariable(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0x0f])
		}
	}
	return b.String()
}
