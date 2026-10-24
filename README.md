![Tests](https://github.com/cdwiegand/go-curling/actions/workflows/tests.yml/badge.svg)
![Docker](https://github.com/cdwiegand/go-curling/actions/workflows/docker.yml/badge.svg)

# Purpose
This program provides a small, Go-based HTTP/HTTPS client for scripts and Docker images that do not ship curl. Copy the static binary into your application image without installing curl and its package dependencies. The published Docker image exists as a source for `COPY --from`; it does not run a service or define a health check. The goal is to support useful scripting features of the [original curl](https://curl.se/) while remaining Go based.

# Examples

```
curl -D - -o - https://google.com
curl -D /dev/null -o /dev/null https://google.not.valid.haha
curl https://google.com
curl https://my.local.test:443 -k
```

# Install

## Binary Release

You can manually download a binary release for Linux, MacOS, Windows or FreeBSD
from the [releases](https://github.com/cdwiegand/go-curling/releases) page.

## Go

Please notice `latest` will install the dev version:

```sh
go install -ldflags="-s -w" -v github.com/cdwiegand/go-curling@latest
```

# Using in a Dockerfile

Copy just the binary into your application image:

```
COPY --from=cdwiegand/go-curling:latest /bin/curl /usr/bin/curl
# OR COPY --from=ghcr.io/cdwiegand/go-curling:latest /bin/curl /usr/bin/curl
```

HTTPS uses the destination image's CA certificates. For images without a trust store, copy a CA bundle into the destination image and pass `--cacert /path/to/ca-bundle.crt`. Private CA files and directories can be supplied with `--cacert` and `--capath`.

## Scripting examples

```sh
# Download with bounded attempts, startup retries and cleanup on failure.
curl -fsS --connect-timeout 2 --max-time 30 \
  --retry 3 --retry-connrefused --retry-max-time 60 \
  --create-dirs --remove-on-error --max-filesize 104857600 \
  -o downloads/archive.tar.gz https://example.com/archive.tar.gz

# Pipe a JSON request without a temporary file.
printf '%s' '{"enabled":true}' | curl -sS --json @- https://example.com/api

# Query HTTP over a local Unix socket.
curl --unix-socket /var/run/docker.sock http://localhost/version
```

# Differences between original curl and go-curling
This program only attempts to support HTTP/HTTPS protocols - others like IMAP, SMTP, RTMP, etc. are not supported.

Not all HTTP-related functionality is supported either, but normal calls like GET, POST, PUT, DELETE, etc. are implemented for the vast majority of use cases, but one difference that makes this not 100% drop-in would be that the `--cookie-jar`/`-c` is both read and write (and `-b`/`--cookie` is "read only"). So normally if you want to use cookies to login a session, just use `--cookie-jar`/`-c` in each call - no need to specify `--cookie`/`-b` unless you want to specify one or more "starting" cookie values.

- Globbing is NOT supported
- Environment variable interpolation ("Variables" in the curl man page) is not supported
- Command line arguments not listed as supported are not supported
- You cannot merge "short form" arguments directly with their values, e.g.: `curl -darbitrary https://...` is not supported, you must use `curl -d arbitrary https://...`
- `no-xxx` form arguments are generally not recognized, unless documented by default their positive version being true (e.g. `--no-fail` doesn't exist as `--fail` is not a default value, but `--no-ca-native` does exist because by default we load the native CA certifications from the underlying OS and so `--ca-native` doesn't exist to turn "on")
- go-curling does not implement global vs scoped arguments - `-:` / `--next` are not supported

Note that one thing that is now supported is that if you specify multiple URLs, you can specify multiple `-o` or `-D` values and go-curling will honor that, but if you specify more URLs than you have specified outputs, the extra URLs will be processed with the default value for the given flag (content output to stdout).

## curl arguments supported
 
| curl argument | supported? | notes |
| -- | -- | -- | 
| `--basic` | (default) | Is only supported auth mech |
| `--ca-native` | (default) | `--no-ca-native` used to turn off |
| `--cacert` | yes | Load trusted PEM certificates; repeatable; `--ca-cert` is a legacy alias |
| `--capath` | yes | Load PEM certificates from a directory; `--ca-path` is a legacy alias |
| `-E`/`--cert` | yes | **(missing tests)** |
| `--compressed` | yes | Request gzip compression and decompress the response |
| `-K`/`--config` | yes | Allows reading config values just like the cli parameters |
| `-b`/`--cookie` | yes | HTTP cookie string or `@`file-path, specifies initial HTTP cookies |
| `-c`/`--cookie-jar` | yes | Specifies file to use for ongoing cookies between requests, cannot use curl's native jar files |
| `-d`/`--data`/`--data-ascii` | yes | Send raw string data name=value OR name=`@`file-path |
| `--data-binary` | yes | Send raw binary data name=value OR name=`@`file-path | |
| `--data-raw` | yes | Send next parameter exactly as given (does not read `@` file value) |
| `--data-urlencode` | yes | Send URL encoded data name=value OR name=`@`file-path |
| `-D`/`--dump-header` | yes | Where to output headers, /dev/null default **(missing tests)** |
| `--expect100-timeout` | yes | Time in decimal seconds to wait for 100-continue header, default 1.0s **(missing tests)** |
| `-f`/`--fail` | yes | On HTTP status >= 400, suppress the body and return exit code 22 |
| `--fail-early` | yes | Fail IMMEDIATELY at error and do not process remaining URLs on command line **(missing tests)** |
| `--fail-with-body` | yes | If fail, will still process output as specified on command line |
| `-F`/`--form` | yes | Send next parameter as a multipart form field (or attach `@file`), name=value OR name=`@`file-path OR name=`<`file-path |
| `--form-string` | yes | Sends parameter as literal value, no `@` or `<` support |
| `-G`/`--get` | yes | Pass -d/--data and related parameters as GET query string parameters instead |
| `-I`/`--head` | yes | Send HEAD request, only emit headers returned, ignore body **(missing tests)** |
| `-H`/`--header` | yes | Header to append to request in the format `"header: value"` |
| `-h`/`--help` | yes | **(missing tests)** |
| `-i`/`--include` | yes | Prepend returned headers to body output **(missing tests)** |
| `-k`/`--insecure` | yes | Ignore invalid SSL certificates **(missing tests)** |
| `--json` | yes | Sends the value as JSON, including setting the content-type appropriately |
| `-j`/`--junk-session-cookies` | yes | Does not store session cookies after all URLs completed **(missing tests)** |
| `--no-keepalive` | yes | Disable keepalive **(missing tests)** |
| `--key` | yes | **(missing tests)** |
| `-L`/`--location` | yes | Allows following redirects to a new location |
| `--location-trusted` | yes | **(missing tests)** |
| `--max-redirs` | yes | **(missing tests)** |
| `--oauth2-bearer` | yes | **(missing tests)** |
| `-o`/`--output` | yes | Where to output results, /dev/stdout default |
| `--pass` | yes | **(missing tests)** |
| `--post301` | yes | **(missing tests)** |
| `--post302` | yes | **(missing tests)** |
| `--post303` | yes | **(missing tests)** |
| `--proto-default` | yes | **(missing tests)** |
| `-e`/`--referer` | yes | HTTP referer header **(missing tests)** |
| `-X`/`--request` | yes | HTTP method to use (generally `GET` unless overridden by other parameters) |
| `-e`/`--referer` | yes | HTTP referer header **(missing tests)** |
| `-e`/`--referer` | yes | HTTP referer header **(missing tests)** |
| `--retry` | yes | Retry transient HTTP errors (408, 429, 500, 502, 503, 504) and timeouts; default exponential backoff |
| `--retry-all-errors` | yes | With `--retry`, also retry other transfer errors and HTTP errors (4xx & 5xx) |
| `--retry-connrefused` | yes | With `--retry`, also retry refused connections |
| `--retry-max-time` | yes | Limit the time available before another retry may start |
| `--retry-delay` | yes | Fixed delay in seconds; zero selects default backoff; honors `Retry-After` |
| `-S`/`--show-error` | yes | Show error info even if silent/fail modes on **(missing tests)** |
| `-s`/`--silent` | yes | Suppress diagnostics; response data and write-out output are still emitted |
| `--stderr` | yes | Log errors, /dev/stderr default |
| `--tls-max` | yes | Force TLS connection max version (1.0, 1.1, 1.2, 1.3, default) **(missing tests)** |
| `-1`/`--tlsv1` | yes | Force TLS connections to at least 1.0 **(missing tests)** |
| `--tlsv1.0` | yes | Force TLS connections to at least 1.0 **(missing tests)** |
| `--tlsv1.1` | yes | Force TLS connections to at least 1.1 **(missing tests)** |
| `--tlsv1.2` | yes | Force TLS connections to at least 1.2 **(missing tests)** |
| `--tlsv1.3` | yes | Force TLS connections to at least 1.3 **(missing tests)** |
| `-T`/`--upload-file` | yes | Upload file(s) to given URL(s) 1:1, as PUT, MIME type detected |
| `--url` | yes | **(missing tests)** |
| `-u`/`--user` | yes | Username:Password for HTTP Basic Authentication **(missing tests)** |
| `-A`/`--user-agent` | yes | User-agent to use (`go-curling/XXXXX` default, XXXXX is a version/build identifier) **(missing tests)** |
| `-v`/`--verbose` | yes | Emit request/response diagnostics to stderr (or `--stderr`) |
| `-V`/`--version` | yes | Return version and exit**(missing tests)** |
| `-w`/`--write-out` | yes | After completion write out the given format to stdout; supports `%{variable}`, `%header{name}`, `%%` and `\n`/`\t`/`\r`; use `@file` or `@-` to load the format from a file/stdin |
| `-m`/`--max-time` | yes | Maximum seconds per attempt, including all redirects and the response body |
| `--connect-timeout` | yes | Maximum time in seconds to allow for establishing the connection |
| `-O`/`--remote-name` | yes | Write output to a local file named like the remote file (from the URL path) |
| `-J`/`--remote-header-name` | yes | With `-O`, prefer the filename from the `Content-Disposition` response header |
| `-r`/`--range` | yes | Request a byte range, sent as a `Range: bytes=<range>` header (e.g. `0-499`) |
| `-4`/`--ipv4` | yes | Resolve names to IPv4 addresses only |
| `-6`/`--ipv6` | yes | Resolve names to IPv6 addresses only |
| `--resolve` | yes | Resolve `HOST:PORT` to a given `ADDRESS` instead of using DNS (repeatable; `HOST` may be `*`) |
| `--http2` | yes | Attempt an HTTP/2 connection |
| `--http1.1` | yes | Use HTTP/1.1 (disables HTTP/2) |
| `-0`/`--http1.0` | yes | Use HTTP/1.0 (disables HTTP/2 and keep-alive; wire version approximated by the Go client) |
| `--unix-socket` | yes | Connect to HTTP/HTTPS through a Unix domain socket; bypasses proxies |
| `-x`/`--proxy` | yes | Override the HTTP/HTTPS proxy; an empty value disables environment proxies |
| `--noproxy` | yes | Comma-separated hosts, domains, IPs or CIDRs to bypass; `*` bypasses all; empty clears inherited exclusions |
| `--create-dirs` | yes | Create parent directories for output files |
| `--remove-on-error` | yes | Remove body output files written by a failed transfer |
| `--max-filesize` | yes | Limit downloaded body bytes, including responses with no Content-Length; 0 is unlimited |
| `-N`/`--no-buffer` | yes | Response bodies are streamed to the destination without whole-body buffering |

# General Arguments Notes

* `--version` is intended to return a build date/version header, and is not intended for parsing by programs. It will return immediately and not process any requests.
* `--request` / `-X` allows you to specify an explicit HTTP verb, some parameters will also inherently override it (ex: `-I`/`--head` will set it to HEAD).
* `--head` / `-I` will suppress content output and will emit the headers to the "content" output location. This means that `--head -o /tmp/1` is the same as `-D /tmp/1 -o /dev/null`.
* `--dump-header` / `-D` will emit the HTTP response headers (if set to `-`, they will appear BEFORE the content output).
* `--output {file path or -}` redirects the content output to another location than stdout.
* `--stderr {file path or -}` will emit errors to the given location.
* `--user-agent {value}` will send the given user agent via HTTP instead of the default.
* `--insecure` / `-k` will ignore invalid HTTPS certificates.
* `--fail` / `-f` will suppress outputs ONLY ON FAILURE and just return a failure error code (non-zero); success will still output by default. 
* `--silent` / `-s` suppresses diagnostics without changing HTTP failure handling or suppressing response data. Use `-o /dev/null` to discard the body. Use `--fail` or `--fail-with-body` to return a failure exit code for HTTP errors; those two flags are mutually exclusive.
* `--show-error` / `-S` will show error info even if silent/fail modes on.
* `--include` / `-i` will include emit returned headers and output to the output path (effectively `-D - -o -`, or `-D file1 -o file1`).
* `--user` allows you to specify a Basic HTTP `username:password` style authentication header.
* `--referer` specifies the `Referer` HTTP header.
* `--header` / `-H` (repeatable) allows you to specify any valid HTTP header, and will override defaults set by other parameters (such as `-d` or `--form`).
* `--cookie` / `-b` (repeatable) allows you to specify an HTTP cookie (as a string, or as a file containing the cookie definition.
* `--cookie-jar` / `-c` allows you to store HTTP cookies for multiple invocations.
* `--post301`, `--post302`, and `--post303` retain POST as the method, along with any file/data uploads on those status codes. Normally we will drop to GET and also drop any data/file arguments.
* `--max-redirs` limits redirects to 50 by default. Zero permits no redirects; -1 permits unlimited redirects. Exceeding the limit returns exit code 47.
* `--proto-default` specifies the default protocol for new URLs (default: http)
* `--oauth2-bearer` specifies an OAuth2 Authorization header (Bearer: xxx) to pass to the first request.
* `--location-trusted` follows redirects and permits them to retain credentials. Otherwise Basic/Bearer authentication and explicitly supplied Authorization/Cookie headers are retained only for the original origin (same scheme, hostname and port). Cookie-jar cookies follow their own domain/path/security rules.
* `--write-out` / `-w` prints a format string to stdout after the transfer completes (always to stdout, and not suppressed by `--silent`). In the format, `%{variable}` is replaced with a value, `%header{name}` with a response header, `%%` is a literal `%`, and `\n` `\t` `\r` are the usual escapes. An unknown `%{variable}` renders as empty. Prefix the whole value with `@` to read the format from a file (`@-` reads stdin). Supported variables: `http_code`/`response_code`, `http_version`, `content_type`, `num_headers`, `num_redirects`, `size_download`, `size_header`, `size_upload`/`size_request`, `url`, `url_effective`, `scheme`, `method`, `remote_ip`, `remote_port`, `local_ip`, `local_port`, `ssl_verify_result`, `exitcode`, `errormsg`, and the timers `time_namelookup`, `time_connect`, `time_appconnect`, `time_pretransfer`, `time_starttransfer`, `time_total`, `time_redirect`. Timers are measured against the final request; for a reused keep-alive connection the connection-phase timers report `0`, and `size_header` is a close approximation of the on-wire size.
* `--max-time` / `-m` bounds each complete attempt (connect + redirects + headers + body). Retries start a fresh attempt timer. `--retry-max-time` prevents another retry when its delay would exhaust the retry budget; an already-started attempt can finish beyond that budget. Use both flags to bound retries and attempts. `--connect-timeout` bounds dialing and the TLS handshake.
* `--retry` uses delays of 1, 2, 4, ... seconds, capped at 600 seconds, unless a nonzero `--retry-delay` is specified. A longer server `Retry-After` value takes precedence. Request bodies are replayed on retries; failed response bodies are closed. Body-read failures can be retried with `--retry-all-errors` (timeouts also qualify with plain `--retry`). Retrying a write operation can repeat server-side effects. Output files are rolled back before a retry, but stdout, pipes and other non-seekable destinations cannot be rolled back and may receive partial or repeated data.
* Response bodies stream to their destination; interrupted transfers return a failure exit code. `--remove-on-error` removes body files touched by the failed transfer, including when `--fail-with-body` reports an HTTP error, and leaves files the transfer never opened alone. Request bodies, including stdin, are buffered in memory for replay.
* Proxy environment variables are inherited from Go's HTTP transport. `--proxy` overrides the HTTP/HTTPS proxy and `--noproxy` overrides exclusions without changing the process environment. `--unix-socket` always bypasses proxies.
* `--remote-name` / `-O` saves the body to a file named after the last path segment of the URL (one per URL); it errors if the URL has no file name. `--remote-header-name` / `-J` prefers the `Content-Disposition` filename when present, with any directory components stripped for safety. `-O`/`-J` override `-o` for the body output.
* `--range` / `-r` sends a `Range: bytes=<range>` header unless you already set a `Range` header via `-H`.
* `--ipv4` / `-4` and `--ipv6` / `-6` force the address family when connecting (mutually exclusive).
* `--resolve HOST:PORT:ADDRESS` dials the given address while keeping the original host for the request and TLS/SNI verification (like curl); `HOST` may be `*` to match any host on that port, and the option is repeatable.
* `--http1.1` and `--http2` select the HTTP version (disabling the other); `--http1.0` / `-0` additionally sends `Connection: close`. Note the Go HTTP client always writes an HTTP/1.1 request line on the wire, so `--http1.0` is an approximation (keep-alive is disabled, but the wire version is not downgraded).

# File/Form/Upload Arguments Notes

* The `--data*` parameters will by default use `POST` as the HTTP verb and `application/x-www-form-urlencoded` as the content type (unless you specify a `Content-Type` header via `-H`):
* `--data-binary @-`, `--json @-` and `--upload-file -` read the request body from stdin. The input is cached for redirects, retries and repeated URL transfers. `--write-out @-` separately consumes stdin as a format string; use a file for one of these inputs if you need both.
* * Using `--data name=value` will send `name` as a raw (already URL encoded) value `value`.
* * Using `--data name=@value` will send `name` as a raw (already URL encoded) value read from the file `value`.
* * Using `--data-binary name=value` will send `name` as a raw (already URL encoded) value `value`.
* * Using `--data-binary name=@value` will send `name` as a raw (already URL encoded) value read from the file `value`.
* * Using `--data-urlencoded name=value` will send `name` as a to-be URL encoded value `value`.
* * Using `--data-urlencoded name=@value` will send `name` as a to-be URL encoded value read from the file `value`.
* * Using `--data-raw name=value` will send `name` as a raw (already URL encoded) value `value` even if it starts with `@`!
* The `--form` parameter will by default use `POST` as the HTTP verb and `multipart/form-data` as the content type (unless you specify a `Content-Type` header via `-H` - strongly discouraged):
* * Using `--form name=value` will send a field `name` with value `value`.
* * Using `--form name=<value` will send a field `name` with value read from `value`.
* * Using `--form name=@value` will send a file `name` with contents read from `value`.
* The `--upload` parameter will by default use `PUT` as the HTTP verb and if possible detect the MIME type using the file extension, or use `application/octet-stream` as the content type (unless you specify a `Content-Type` header via `-H`):
* * Using `--upload value` will send the contents of the `value` file as the entire body.

# Error Codes

Exit codes are positive and portable across shells. This replaces the previous negative codes (which wrapped to values such as 250 on Unix).

- 0: Transfer succeeded; HTTP errors also return 0 unless `--fail` or `--fail-with-body` is used
- 2: Invalid arguments or initialization failure
- 3: Invalid/missing URL or unsupported scheme
- 6: Unable to resolve host
- 7: Connection/request failure
- 18: Incomplete response body
- 22: HTTP status >= 400 with `--fail` or `--fail-with-body`
- 23: Unable to write output, headers, cookies, stdout or stderr
- 26: Unable to read request data/upload file
- 28: Operation timed out
- 47: Redirect limit exceeded
- 56: Error receiving the response body
- 60: TLS certificate verification/system trust failure
- 63: Response exceeds `--max-filesize`
- 77: Unable to load a requested CA file/directory

`%{exitcode}` and `%{errormsg}` in `--write-out` describe the completed transfer, including HTTP, timeout and output errors. A failure writing the write-out text itself is returned by the process exit code.

# Tests
Tests are now present in the code - run `go test -v ./...` to run them. Most test files contain both 

# License
go-curling is licensed under the [MIT License](./LICENSE). Previously it was licensed under the LGPL - as I am the sole author prior to this change, I approve the change.

# Credits

Lots of credit to the [original authors of curl](https://curl.se/docs/thanks.html), as well as to [@emacampolo](https://github.com/emacampolo) for a great JSON comparator class, [@ericbsantana](https://github.com/ericbsantana) for [gurl](https://github.com/ericbsantana/gurl) that inspired me to do more with a simple project, and everyone else who has posted golang code on the web for the rest of us to learn from!

# curl arguments not supported yet

- `--abstract-unix-socket`
- `--alt-svc`
- `--anyauth`
- `--aws-sigv4`
- `--cert-status`
- `--cert-type `
- `--ciphers`
- `--connect-to`
- `-C`/`--continue-at`
- `--create-file-mode`
- `--crlf`
- `--crlfile`
- `--curves`
- `--delegation`
- `--digest`
- `-q`/`--disable`
- `--disallow-username-in-url`
- `--dns-interface`
- `--dns-ipv4-addr`
- `--dns-ipv6-addr`
- `--dns-servers`
- `--doh-cert-status`
- `--doh-insecure`
- `--doh-url`
- `--ech`
- `--egd-file`
- `--engine`
- `--etag-compare`
- `--etag-save`
- `--false-start`
- `--form-escape`
- `-g`/`--globoff`
- `--happy-eyeballs-timeout-ms`
- `--haproxy-clientip`
- `--haproxy-protocol`
- `--hsts`
- `--http0.9`
- `--http2-prior-knowledge`
- `--http3`
- `--http3-only`
- `--ignore-content-length`
- `--interface`
- `--ipfs-gateway`
- `--keepalive-time`
- `--key-type`
- `--krb`
- `--libcurl`
- `--limit-rate`
- `--local-port`
- `-M`/`--manual`
- `--metalink`
- `--negotiate`
- `-n`/`--netrc`
- `--netrc-file`
- `--netrc-optional`
- `-:`/`--next`
- `--no-alpn`
- `--no-clobber`
- `--no-npn`
- `--no-progress-meter`
- `--no-sessionid`
- `--ntlm`
- `--output-dir`
- `-Z`/`--parallel`
- `--parallel-immediate`
- `--parallel-max`
- `--path-as-is` *`go-curling` does not modify given URL(s)*
- `--pinnedpubkey`
- `-#`/`--progress-bar`
- `--rate`
- `--raw`
- `--remote-name-all`
- `-R`/`--remote-time`
- `--request-target`
- `--service-name`
- `-Y`/`--speed-limit`
- `-y`/`--speed-time`
- `--ssl`
- `--ssl-allow-beast`
- `--ssl-auto-client-cert`
- `--ssl-no-revoke`
- `--ssl-reqd`
- `--ssl-revoke-best-effort`
- `-2`/`--sslv2`
- `-3`/`--sslv3`
- `--styled-output`
- `--tcp-fastopen`
- `--tcp-nodelay` *Need to add `no-tcp-nodelay`*
- `-z`/`--time-cond`
- `--tls13-ciphers`
- `--tlsauthtype`
- `--tlspassword`
- `--tlsuser`
- `--tr-encoding`
- `--trace`
- `--trace-ascii`
- `--trace-config`
- `--trace-ids`
- `--trace-time`
- `--url-query`
- `--variable`
- `--xattr`

These are not applicable because `go-curling` does not support proxies yet:

- `--no-proxy` 
- `--preproxy` 
- `--proxy-anyauth` 
- `--proxy-basic` 
- `--proxy-ca-native` 
- `--proxy-cacert` 
- `--proxy-capath` 
- `--proxy-cert` 
- `--proxy-cert-type` 
- `--proxy-ciphers` 
- `--proxy-crlfile` 
- `--proxy-digest` 
- `--proxy-header` 
- `--proxy-http2` 
- `--proxy-insecure` 
- `--proxy-key` 
- `--proxy-key-type` 
- `--proxy-negotiate` 
- `--proxy-ntlm` 
- `--proxy-pass` 
- `--proxy-pinnedpubkey` 
- `--proxy-service-name` 
- `--proxy-ssl-allow-beast` 
- `--proxy-ssl-auto-client-cert` 
- `--proxy-tls13-ciphers` 
- `--proxy-tlsauthtype` 
- `--proxy-tlspassword` 
- `--proxy-tlsuser` 
- `--proxy-tlsv1` 
- `--socks4` 
- `--socks4a` 
- `--socks5` 
- `--socks5-basic` 
- `--socks5-gssapi` 
- `--socks5-gssapi-nec` 
- `--socks5-gssapi-service` 
- `--socks5-hostname` 

## curl arguments not applicable

These are not applicable because they are only for protocols other than HTTP/S or they are deprecated in upstream `curl`:

- `-a`/`--append`
- `--compressed-ssh`
- `-q`/`--disable`
- `--disable-eprt`/`--no-eprt`
- `--eprt`
- `--disable-epsv,--no-epsv`
- `--epsv`
- `--ftp-account`
- `--ftp-alternative-to-user`
- `--ftp-create-dirs`
- `--ftp-method`
- `--ftp-pasv`
- `-P`/`--ftp-port`
- `--ftp-pret`
- `--ftp-skip-pasv-ip`
- `--ftp-ssl-ccc`
- `--ftp-ssl-ccc-mode`
- `--ftp-ssl-control`
- `--hostpubmd5`
- `--hostpubsha256`
- `-l`/`--list-only`
- `--login-options`
- `--mail-auth`
- `--mail-from`
- `--mail-rcpt`
- `--mail-rcpt-allowfails`
- `--ntlm-wb` *Deprecated in curl*
- `--proto`
- `--proto-redir`
- `-x`/`--proxy`
- `--pubkey`
- `-Q`/`--quote`
- `--random-file` *Deprecated in curl*
- `--sasl-authzid`
- `--sasl-ir`
- `-t`/`--telnet-option`
- `--tftp-blksize`
- `--tftp-no-options`
- `-B`/`--use-ascii`
