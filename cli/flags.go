package cli

import (
	"bufio"
	"os"
	"strings"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	flag "github.com/spf13/pflag"
)

type ParseOnlyArgs struct {
	ConfigFile string
}

func SetupConfigParseOnlyArgs(parseCtx *ParseOnlyArgs, flags *flag.FlagSet) {
	flags.StringVarP(&parseCtx.ConfigFile, "config", "K", "", "Config file to parse for go-curling / curl")
}

func SetupFlagArgs(ctx *curl.CurlContext, flags *flag.FlagSet) {
	empty := []string{}
	flags.BoolVarP(&ctx.Version, "version", "V", false, "Return version and exit")
	flags.BoolVarP(&ctx.Verbose, "verbose", "v", false, "Logs all headers, and body to output")
	flags.StringVar(&ctx.ErrorOutput, "stderr", curl.DEFAULT_STDERR, "Log errors to this replacement for stderr")
	flags.StringVarP(&ctx.HttpVerb, "request", "X", "", "HTTP method to use (usually GET unless otherwise modified by other parameters)")
	flags.StringArrayVarP(&ctx.BodyOutput, "output", "o", []string{curl.DEFAULT_OUTPUT}, "Where to output results")
	flags.StringArrayVarP(&ctx.HeaderOutput, "dump-header", "D", []string{}, "Where to output headers (not on by default)")
	flags.StringVarP(&ctx.UserAgent, "user-agent", "A", "go-curling/##DEV##", "User-agent to use")
	flags.StringVarP(&ctx.UserAuth, "user", "u", "", "User:password for HTTP authentication")
	flags.StringVarP(&ctx.Referer, "referer", "e", "", "Referer URL to use with HTTP request")
	flags.StringArrayVar(&ctx.Urls, "url", []string{}, "Requesting URL")
	flags.BoolVarP(&ctx.SilentFail, "fail", "f", false, "If fail do not emit contents just return fail exit code (-6)")
	flags.BoolVar(&ctx.FailEarly, "fail-early", false, "If any URL fails, stop immediately and do not continue.")
	flags.BoolVar(&ctx.FailWithBody, "fail-with-body", false, "If fail emit contents and return fail exit code (-6)")
	flags.BoolVarP(&ctx.IgnoreBadCerts, "insecure", "k", false, "Ignore invalid SSL certificates")
	flags.BoolVarP(&ctx.IsSilent, "silent", "s", false, "Silence all program console output")
	flags.BoolVarP(&ctx.ShowErrorEvenIfSilent, "show-error", "S", false, "Show error info even if silent mode on")
	flags.BoolVarP(&ctx.HeadOnly, "head", "I", false, "Only return headers (ignoring body content)")
	flags.BoolVarP(&ctx.IncludeHeadersInMainOutput, "include", "i", false, "Include headers (prepended to body content)")
	flags.StringSliceVarP(&ctx.Cookies, "cookie", "b", empty, "HTTP cookie, raw HTTP cookie only (use -c for cookie jar files)")
	flags.StringSliceVarP(&ctx.Data_Standard, "data", "d", empty, "HTML form data (send as-is, except for @file reference strips strips CR/LF), sets mime type to 'application/x-www-form-urlencoded' unless specified as a header")
	flags.StringSliceVar(&ctx.Data_Encoded, "data-urlencode", empty, "HTML form data (value or @file lines are URL encoded), sets mime type to 'application/x-www-form-urlencoded' unless specified as a header")
	flags.StringSliceVar(&ctx.Data_RawAsIs, "data-raw", empty, "HTML form data (send value exactly as-is, no @file support), sets mime type to 'application/x-www-form-urlencoded' unless specified as a header")
	flags.StringSliceVar(&ctx.Data_Binary, "data-binary", empty, "HTML form data (send value exactly as-is, except for @file reference), sets mime type to 'application/x-www-form-urlencoded' unless specified as a header")
	flags.StringArrayVar(&ctx.Data_Json, "json", empty, "Submits data already in JSON format, sets mime type to 'application/json' unless specified as a header, @file is supported")
	flags.StringSliceVarP(&ctx.Form_Multipart, "form", "F", empty, "HTML form data (multipart MIME), sets mime type to 'multipart/form-data' unless specified as a header")
	flags.StringSliceVar(&ctx.Form_MultipartRaw, "form-string", empty, "HTML form data (multipart MIME), exact value used, no @file or >file support")
	flags.StringVarP(&ctx.CookieJar, "cookie-jar", "c", "", "File for storing (read and write) cookies")
	flags.BoolVar(&ctx.JunkSessionCookies, "junk-session-cookies", false, "Does not store session cookies in cookie jar")
	flags.StringArrayVarP(&ctx.Upload_File, "upload-file", "T", []string{}, "Raw file(s) to PUT (default) to the url(s) given, not encoded, sets mime type to detected mime type for extension unless specified as a header")
	flags.StringArrayVarP(&ctx.Headers, "header", "H", []string{}, "Header(s) to append to request")
	flags.BoolVar(&ctx.DoNotUseHostCertificateAuthorities, "no-ca-native", false, "Do not use the host's Certificate Authorities (turns off --ca-native)")
	flags.StringArrayVar(&ctx.CaCertFile, "ca-cert", nil, "Specifies PEM file(s) containing certs for trusted Certificate Authorities")
	flags.StringVar(&ctx.CaCertPath, "ca-path", "", "Specifies a directory container PEM files containing certs for trusted Certificate Authorities")
	flags.StringVarP(&ctx.ClientCertFile, "cert", "E", "", "Client certificate (cert or cert + key) to use for authentication to server, with :password after if key is encrypted")
	flags.StringVar(&ctx.ClientCertKeyFile, "key", "", "Client certificate key to use for authentication to server, with :password after if encrypted")
	flags.StringVar(&ctx.ClientCertKeyPassword, "key-password", "", "Password to decrypt client certificate key") // NOT UPSTREAM curl!
	flags.BoolVar(&ctx.EnableCompression, "compressed", false, "Requests compression")
	//flags.BoolVar(&ctx.EnableCompression, "tr-encoding", false, "Requests compression (obsolete)")
	//flags.MarkHidden("tr-encoding")
	flags.BoolVar(&ctx.DisableKeepalives, "no-keepalive", false, "Disable use of keepalive messages")
	flags.BoolVarP(&ctx.DisableBuffer, "no-buffer", "N", false, "Disables buffering")
	flags.BoolVarP(&ctx.FollowRedirects, "location", "L", false, "Follow redirects (3xx response Location headers)")
	flags.IntVar(&ctx.MaxRedirects, "max-redirs", 50, "Maximum 3xx redirects to follow before stopping")
	flags.StringVar(&ctx.DefaultProtocolScheme, "proto-default", "http", "Specifies default protocol to prepend to URLs")
	flags.BoolVar(&ctx.Allow301Post, "post301", false, "If 301 redirect returned do not change method (to GET)")
	flags.BoolVar(&ctx.Allow302Post, "post302", false, "If 302 redirect returned do not change method (to GET)")
	flags.BoolVar(&ctx.Allow303Post, "post303", false, "If 303 redirect returned do not change method (to GET)")
	flags.StringVar(&ctx.OAuth2_BearerToken, "oauth2-bearer", "", "OAuth2 Authorization header (Bearer: xxx)")
	flags.BoolVar(&ctx.RedirectsKeepAuthenticationHeaders, "location-trusted", false, "Allow redirects to also receive Authentication headers")
	flags.StringVarP(&ctx.ConfigFile, "config", "K", "", "Config file to pre-configure go-curling")
	flags.BoolVar(&ctx.Tls_MinVersion_1_3, "tlsv1.3", false, "Force TLS connections to version 1.3 or higher")
	flags.BoolVar(&ctx.Tls_MinVersion_1_2, "tlsv1.2", false, "Force TLS connections to version 1.2 or higher")
	flags.BoolVar(&ctx.Tls_MinVersion_1_1, "tlsv1.1", false, "Force TLS connections to version 1.1 or higher")
	flags.BoolVar(&ctx.Tls_MinVersion_1_0, "tlsv1.0", false, "Force TLS connections to version 1.0 or higher")
	flags.BoolVarP(&ctx.Tls_MinVersion_1_0, "tlsv1", "1", false, "Force TLS connections to version 1.0 or higher")
	flags.StringVar(&ctx.Tls_MaxVersionString, "tls-max", "", "Force TLS connections to maximum version specified")
	// flags.BoolVar(&ctx.ForceTryHttp2, "http2", false, "Force trying an HTTP2 connection initially")
	flags.BoolVarP(&ctx.ConvertPostFormIntoGet, "get", "G", false, "Convert -d/--data and related parameters into GET query string parameters")
}

func ParseFlags(args []string, ctx *curl.CurlContext) ([]string, *curlerrors.CurlError) {
	// args: os.Args[1:] normally, if testing you provide :)
	// I want to be able to test using my own args[], so can't use default flag.Parse()..

	for i := 0; i < len(args); i++ {
		if args[i] == "-K" || args[i] == "--config" {
			moreArgs, err2 := ParseConfigFile(args[i+1])
			if err2 != nil {
				return nil, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_INVALID_ARGS, "Invalid args/failed to parse flags", err2)
			}
			args = append(args, moreArgs...)
		}
	}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	SetupFlagArgs(ctx, flags)
	err := flags.Parse(args)
	extraArgs := flags.Args() // remaining non-parsed args
	if err != nil {
		return nil, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_INVALID_ARGS, "Invalid args/failed to parse flags", err)
	}
	return extraArgs, nil
}

func ParseConfigFile(path string) ([]string, error) {
	// each line is separate "arguments", with some tweaks, examples below

	/*
	  # short form no arg permitted as-is
	  -s
	  # short form with args too
	  -o /dev/null
	  # long form no arg can have --
	  --post301
	  # long form no arg also can omit --
	  post302
	  # long forms with arg the same..
	  --oauth2-bearer Testing1234
	  oauth2-bearer Testing1234
	  oauth2-bearer "this is a test"
	  # all of these valid, if you omit the -- you can use this format
	  oauth2-bearer = "this is a test"
	  oauth2-bearer: "this is a test"
	  oauth2-bearer ="this is a test"
	  oauth2-bearer : "this is a test"
	  oauth2-bearer="this is a test"
	  oauth2-bearer:"this is a test"
	*/

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ret []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		words := ParseConfigLine(line)
		if len(words) > 0 {
			ret = append(ret, words...)
		}
	}

	if err := scanner.Err(); err != nil {
		return ret, err
	}

	return ret, nil
}

func ParseConfigLine(line string) []string {
	line = strings.TrimSpace(line)
	if len(line) == 0 || strings.Index(line, "#") == 0 {
		return []string{} // nothing
	}

	if line[0] != '-' {
		// long form w/o the --, a little more work is required
		firstSpace := strings.IndexAny(line, " =:")
		if firstSpace > 0 {
			word1 := "--" + strings.TrimRightFunc(line[0:firstSpace], TrimBoundary)
			word2 := strings.TrimLeftFunc(line[firstSpace+1:], TrimBoundary)
			word2 = strings.Trim(word2, "\"")
			return []string{word1, word2}
		} else {
			return []string{"--" + line}
		}
	} else {
		idxSpace := strings.Index(line, " ")
		if idxSpace > 0 {
			return []string{line[0:idxSpace], line[idxSpace+1:]}
		} else {
			return []string{line}
		}
	}
}

func TrimBoundary(r rune) bool {
	return r == ' ' || r == '=' || r == ':'
}
