![Tests](https://github.com/cdwiegand/go-curling/actions/workflows/tests.yml/badge.svg)
![Docker](https://github.com/cdwiegand/go-curling/actions/workflows/docker.yml/badge.svg)

# Purpose
This program was designed to replace the curl that is no longer shipped with Microsoft's dotnet core docker containers. Removing that kept breaking all of my upgraded containers, and I really wanted curl back for healthchecks without having to `apt install` and `apt clean` and cleaning out the cache. So I built a simple curl that handled the healthcheck calls I was doing. I have since started expanding it to meet more needs of the [original curl](https://curl.se/), while remaining golang based.

# Examples

```
curl -D - -o - https://google.com
curl -D /dev/null -o /dev/null https://google.not.valid.haha
curl https://google.com
curl https://my.local.test:443 -k
```

# Using in a Dockerfile
```
COPY --from=cdwiegand/go-curling:latest /bin/curl /usr/bin/curl
# OR COPY --from=ghcr.io/cdwiegand/go-curling:latest /bin/curl /usr/bin/curl
HEALTHCHECK CMD curl -s http://localhost:80
```

# Differences between original curl and go-curling
This program only attempts to support HTTP/HTTPS protocols - others like IMAP, SMTP, RTMP, etc. are not supported.

Not all HTTP-related functionality is supported either, but normal calls like GET, POST, PUT, DELETE, etc. are implemented for the vast majority of use cases, but one difference that makes this not 100% drop-in would be that the `--cookie-jar`/`-c` is both read and write. The `--cookie` / `-b` command only specifies a raw HTTP cookie on the command line - it is not usable as a file to read a prior cookie jar, due to the custom JSON format for storing cookies. So normally if you want to use cookies to login a session, just use `--cookie-jar`/`-c` in each call - no need to specify `--cookie`/`-b` unless you want to specify one or more "starting" cookie values.

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
| `--cacert` | yes | **(missing test)** |
| `--capath` | yes | **(missing tests)** Loads all files in path and attempts to parse |
| `-E`/`--cert` | yes | **(missing tests)** |
| `--compressed` | (default) | turn off via `--no-compressed` |
| `-K`/`--config` | yes | Allows reading config values just like the cli parameters |
| `-b`/`--cookie` | yes | HTTP cookie string or `@`file-path, specifies initial HTTP cookies |
| `-c`/`--cookie-jar` | yes | Specifies file to use for ongoing cookies between requests, cannot use curl's native jar files |
| `-d`/`--data` | yes | Send raw string data name=value OR name=`@`file-path |
| `--data-binary` | yes | Send raw binary data name=value OR name=`@`file-path | |
| `--data-raw` | yes | Send next parameter exactly as given (does not read `@` file value) |
| `--data-urlencode` | yes | Send URL encoded data name=value OR name=`@`file-path |
| `-D`/`--dump-header` | yes | Where to output headers, /dev/null default **(missing tests)** |
| `--expect100-timeout` | yes | Time in decimal seconds to wait for 100-continue header, default 1.0s **(missing tests)** |
| `-f`/`--fail` | yes | If fail do not emit contents **(missing tests)** |
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
| `--retry` | yes | Retry X times on specific HTTP errors (408, 429, 500, 502, 503, 504) **(missing tests)** |
| `--retry-all` | yes | Retry any HTTP error (4xx & 5xx) **(missing tests)** |
| `--retry-delay` | yes | Retry after X seconds on failures handled by `--retry` **(missing tests)** |
| `-S`/`--show-error` | yes | Show error info even if silent/fail modes on **(missing tests)** |
| `-s`/`--silent` | yes | Do not emit any output (unless overridden with `show-error`) **(missing tests)** |
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
| `-v`/`--verbose` | yes | **(missing tests)** |
| `-V`/`--version` | yes | Return version and exit**(missing tests)** |

# General Arguments Notes

* `--version` is intended to return a build date/version header, and is not intended for parsing by programs. It will return immediately and not process any requests.
* `--request` / `-X` allows you to specify an explicit HTTP verb, some parameters will also inherently override it (ex: `-I`/`--head` will set it to HEAD).
* `--head` / `-I` will suppress content output and will emit the headers to the "content" output location. This means that `--head -o /tmp/1` is the same as `-D /tmp/1 -o /dev/null`.
* `--dump-header` / `-H` will emit the HTTP response headers (if set to `-`, they will appear BEFORE the content output).
* `--output {file path or -}` redirects the content output to another location than stdout.
* `--stderr {file path or -}` will emit errors to the given location.
* `--user-agent {value}` will send the given user agent via HTTP instead of the default.
* `--insecure` / `-K` will ignore invalid HTTPS certificates.
* `--fail` / `-f` will suppress outputs ONLY ON FAILURE and just return a failure error code (non-zero); success will still output by default. 
* `--silent` / `-s` will not emit any output regardless of success or failure.
* `--show-error` / `-S` will show error info even if silent/fail modes on.
* `--include` / `-i` will include emit returned headers and output to the output path (effectively `-D - -o -`, or `-D file1 -o file1`).
* `--user` allows you to specify a Basic HTTP `username:password` style authentication header.
* `--referer` specifies the `Referer` HTTP header.
* `--header` / `-H` (repeatable) allows you to specify any valid HTTP header, and will override defaults set by other parameters (such as `-d` or `--form`).
* `--cookie` / `-b` (repeatable) allows you to specify an HTTP cookie (as a string, or as a file containing the cookie definition.
* `--cookie-jar` / `-c` allows you to store HTTP cookies for multiple invocations.
* `--post301`, `--post302`, and `--post303` retain POST as the method, along with any file/data uploads on those status codes. Normally we will drop to GET and also drop any data/file arguments.
* `--max-redirs` limits the number of redirections to process to 50 by default. Pass -1, 0, or any negative number to allow unlimited redirects.
* `--proto-default` specifies the default protocol for new URLs (default: http)
* `--oauth2-bearer` specifies an OAuth2 Authorization header (Bearer: xxx) to pass to the first request.
* `--location-trusted` permits redirects to retain authorization headers (basic auth or oauth2 bearer)

# File/Form/Upload Arguments Notes

* The `--data*` parameters will by default use `POST` as the HTTP verb and `application/x-www-form-urlencoded` as the content type (unless you specify a `Content-Type` header via `-H`):
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
- 6: Response present, but a status code >= 400 (e.g. failing) was returned
- 7: No response, but an error was thrown
- 8: Invalid/missing URL
- 9: Unable to read upload file
- 10: Unable to write output file (cookies or output)
- 11: Unable to write to stdout/stderr
- 249: No such host or invalid scheme
- 250: Invalid/missing url

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
- `--connect-timeout`
- `--connect-to`
- `-C`/`--continue-at`
- `--create-dirs`
- `--create-file-mode`
- `--crlf`
- `--crlfile`
- `--curves`
- `--data-ascii`
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
- `-0`/`--http1.0`
- `--http1.1`
- `--http2`
- `--http2-prior-knowledge`
- `--http3`
- `--http3-only`
- `--ignore-content-length`
- `--interface`
- `--ipfs-gateway`
- `-4`/`--ipv4`
- `-6`/`--ipv6`
- `--keepalive-time`
- `--key-type`
- `--krb`
- `--libcurl`
- `--limit-rate`
- `--local-port`
- `-M`/`--manual`
- `--max-filesize`
- `-m`/`--max-time`
- `--metalink`
- `--negotiate`
- `-n`/`--netrc`
- `--netrc-file`
- `--netrc-optional`
- `-:`/`--next`
- `--no-alpn`
- `-N`/`--no-buffer`
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
- `-x`/`--proxy`
- `-r`/`--range`
- `--rate`
- `--raw`
- `-J`/`--remote-header-name`
- `-O`/`--remote-name`
- `--remote-name-all`
- `-R`/`--remote-time`
- `--remove-on-error`
- `--request-target`
- `--resolve`
- `--retry-all-errors`
- `--retry-connrefused`
- `--retry-max-time`
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
- `--unix-socket`
- `--url-query`
- `--variable`
- `-w`/`--write-out`
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