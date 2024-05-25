package cli

import (
	"os"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	"github.com/spf13/pflag"
	flag "github.com/spf13/pflag"
)

func SetupFlagArgs(ctx *curl.CurlContext, flags *flag.FlagSet) {
	empty := []string{}
	flags.BoolVarP(&ctx.Version, "version", "V", false, "Return version and exit")
	flags.BoolVarP(&ctx.Verbose, "verbose", "v", false, "Logs all headers, and body to output")
	flags.StringVar(&ctx.ErrorOutput, "stderr", "stderr", "Log errors to this replacement for stderr")
	flags.StringVarP(&ctx.HttpVerb, "request", "X", "", "HTTP method to use (usually GET unless otherwise modified by other parameters)")
	flags.StringArrayVarP(&ctx.BodyOutput, "output", "o", []string{curl.DEFAULT_OUTPUT}, "Where to output results")
	flags.StringArrayVarP(&ctx.HeaderOutput, "dump-header", "D", []string{}, "Where to output headers (not on by default)")
	flags.StringVarP(&ctx.UserAgent, "user-agent", "A", "go-curling/##DEV##", "User-agent to use")
	flags.StringVarP(&ctx.UserAuth, "user", "u", "", "User:password for HTTP authentication")
	flags.StringVarP(&ctx.Referer, "referer", "e", "", "Referer URL to use with HTTP request")
	flags.StringArrayVar(&ctx.Urls, "url", []string{}, "Requesting URL")
	flags.BoolVarP(&ctx.SilentFail, "fail", "f", false, "If fail do not emit contents just return fail exit code (-6)")
	flags.BoolVar(&ctx.FailEarly, "fail-early", false, "If any URL fails, stop immediately and do not continue.")
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
	flags.BoolVar(&ctx.DisableCompression, "no-compressed", false, "Disables compression")
	flags.BoolVarP(&ctx.FollowRedirects, "location", "L", false, "Follow redirects (3xx response Location headers)")
}

func ParseFlags(args []string, ctx *curl.CurlContext) (*pflag.FlagSet, []string, *curlerrors.CurlError) {
	// args: os.Args[1:] normally, if testing you provide :)
	// I want to be able to test using my own args[], so can't use default flag.Parse()..

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	SetupFlagArgs(ctx, flags)
	err := flags.Parse(args)
	extraArgs := flags.Args() // remaining non-parsed args
	if err != nil {
		return flags, extraArgs, curlerrors.NewCurlError2(curlerrors.ERROR_INVALID_ARGS, "Invalid args/failed to parse flags", err)
	}
	return flags, extraArgs, nil
}
