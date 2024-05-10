package cli

import (
	curl "github.com/cdwiegand/go-curling/context"
	flag "github.com/spf13/pflag"
)

func SetupFlagArgs(ctx *curl.CurlContext, flags *flag.FlagSet) {
	empty := []string{}
	flags.BoolVarP(&ctx.Version, "version", "V", false, "Return version and exit")
	flags.BoolVarP(&ctx.Verbose, "verbose", "v", false, "Logs all headers, and body to output")
	flags.StringVar(&ctx.ErrorOutput, "stderr", "stderr", "Log errors to this replacement for stderr")
	flags.StringVarP(&ctx.Method, "method", "X", "", "HTTP method to use (usually GET unless otherwise modified by other parameters)")
	flags.StringArrayVarP(&ctx.Output, "output", "o", []string{curl.DEFAULT_OUTPUT}, "Where to output results")
	flags.StringArrayVarP(&ctx.HeaderOutput, "dump-header", "D", []string{}, "Where to output headers (not on by default)")
	flags.StringVarP(&ctx.UserAgent, "user-agent", "A", "go-curling/##DEV##", "User-agent to use")
	flags.StringVarP(&ctx.UserAuth, "user", "u", "", "User:password for HTTP authentication")
	flags.StringVarP(&ctx.Referer, "referer", "e", "", "Referer URL to use with HTTP request")
	flags.StringArrayVar(&ctx.Urls, "url", []string{}, "Requesting URL")
	flags.BoolVarP(&ctx.SilentFail, "fail", "f", false, "If fail do not emit contents just return fail exit code (-6)")
	flags.BoolVarP(&ctx.IgnoreBadCerts, "insecure", "k", false, "Ignore invalid SSL certificates")
	flags.BoolVarP(&ctx.IsSilent, "silent", "s", false, "Silence all program console output")
	flags.BoolVarP(&ctx.ShowErrorEvenIfSilent, "show-error", "S", false, "Show error info even if silent mode on")
	flags.BoolVarP(&ctx.HeadOnly, "head", "I", false, "Only return headers (ignoring body content)")
	flags.BoolVarP(&ctx.IncludeHeadersInMainOutput, "include", "i", false, "Include headers (prepended to body content)")
	flags.StringSliceVarP(&ctx.Cookies, "cookie", "b", empty, "HTTP cookie, raw HTTP cookie only (use -c for cookie jar files)")
	flags.StringSliceVarP(&ctx.Data_standard, "data", "d", empty, "HTML form data (raw or @file), set mime type to 'application/x-www-form-urlencoded'")
	flags.StringSliceVar(&ctx.Data_encoded, "data-urlencode", empty, "HTML form data (URL encoded), set mime type to 'application/x-www-form-urlencoded'")
	flags.StringSliceVar(&ctx.Data_rawconcat, "data-raw", empty, "HTML form data (force raw), set mime type to 'application/x-www-form-urlencoded'")
	flags.StringSliceVarP(&ctx.Data_multipart, "form", "F", empty, "HTML form data (multipart MIME), set mime type to 'multipart/form-data'")
	flags.StringVarP(&ctx.CookieJar, "cookie-jar", "c", "", "File for storing (read and write) cookies")
	flags.StringArrayVarP(&ctx.UploadFile, "upload-file", "T", []string{}, "Raw file(s) to PUT (default) to the url(s) given, not encoded")
	flags.StringArrayVarP(&ctx.Headers, "header", "H", []string{}, "Header(s) to append to request")
}
