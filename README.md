# Purpose
This program was designed to replace the curl that is no longer shipped with Microsoft's dotnet core docker containers. Removing that kept breaking all of my upgraded containers, and I really wanted curl back for healthchecks without having to `apt install` and `apt clean` and cleaning out the cache. So I built a simple curl that handled the healthcheck calls I was doing. I have since started expanding it to meet more needs of the [original curl](https://curl.se/), while remaining golang based.

# Differences between original curl and go-curling
This program only attempts to support HTTP/HTTPS protocols - others like IMAP, SMTP, RTMP, etc. are not supported.

Not all HTTP-related functionality is supported either, but normal calls like GET, POST, PUT, DELETE, etc. are implemented for the vast majority of use cases, but one difference that makes this not 100% drop-in would be that the `--cookie-jar`/`-c` is both read and write. The `--cookie` / `-b` command only specifies a raw HTTP cookie on the command line - it is not usable as a file to read a prior cookie jar, due to the custom JSON format for storing cookies. So normally if you want to use cookies to login a session, just use `--cookie-jar`/`-c` in each call - no need to specify `--cookie`/`-b` unless you want to specify one or more "starting" cookie values.

- Globbing is NOT supported
- Environment variable interpolation ("Variables" in the curl man page) is not supported
- Command line arguments not listed below are also not supported
- go-curling does not implement global vs scoped arguments - `-:` / `--next` is not supported

Note that one thing that is now supported is that if you specify multiple URLs, you can specify multiple `-o` or `-D` values and go-curling will honor that, but if you specify more URLs than you have specified outputs, the extra URLs will be processed with the default value for the given flag (content output to stdout).

# Arguments
| short | long form | default | type | description |
| -- | -- | -- | -- | -- |
| `-V` | `--version` | (none) | (none) | Return version and exit |
| `-X` | `--method` | `GET` | string | HTTP method to use (generally `GET` unless overridden by other parameters) |
| `-o` | `--output` | `-` (/dev/stdout) | `-` or file-path(s) | Where to output results |
| `-D` | `--dump-header` | `/dev/null` | `-` or file-path(s) | Where to output headers separately |
|      | `--stderr` | `/dev/stderr` | `-` or file-path | Log errors to this replacement for stderr |
| `-A` | `--user-agent` | `go-curling/1` | string | User-agent to use |
| `-k` | `--insecure` | (false) | flag | Ignore invalid SSL certificates |
| `-f` | `--fail` | (false) | flag | If fail do not emit contents just return fail exit code |
| `-s` | `--silent` | (false) | flag | Do not emit any output (unless overridden with `show-error`) |
| `-S` | `--show-error` | (false) | flag | Show error info even if silent/fail modes on |
| `-i` | `--include` | (false) | flag | Prepend returned headers to body output |
| `-I` | `--head` | (false) | flag | Only emit headers returned, ignore body |
| `-u` | `--user` | (none) | string | Username:Password for HTTP Basic Authentication |
| `-e` | `--referer` | (none) | URI | HTTP referer header |
| `-H` | `--header` | (none) | Header to append to request in the format `"header: value"` | 
| `-b` | `--cookie` | (none) | HTTP cookie string or `@`file-path | Specifies initial HTTP cookies |
| `-c` | `--cookie-jar` | (none) | file-path | Specifies file to which to write ongoing cookies to |
|      | `--junk-session-cookies` | (false) | flag | Does not store session cookies after all URLs completed |
| `-d` | `--data` | (none) | name=value OR name=`@`file-path | Send next parameter as raw string data |
|      | `--data-binary` | (none) | name=value OR name=`@`file-path | Send next parameter raw binary data |
|      | `--data-raw` | (none) | name=value OR name=`@`file-path | Send next parameter exactly as given |
|      | `--data-urlencoded` | (none) | name=value OR name=`@`file-path | Send next parameter URL encoding first |
| `-F` | `--form` | (none) | name=value OR name=`@`file-path OR name=`<`file-path | Send next parameter as a multipart form field (or `@file`) |
| `-T` | `--upload-file` | (none) | file-path | Upload file(s) to given URL(s) 1:1, as PUT, MIME type detected |

# General Arguments

* `--version` is intended to return a build date/version header, and is not intended for parsing by programs. It will return immediately and not process any requests.
* `--method` allows you to specify an explicit HTTP verb, some parameters will also inherently override it (ex: `-I`/`--head` will set it to HEAD).
* `--output {file path or -}` redirects the content output to another location than stdout.
* `--dump-header` will emit the HTTP response headers (if set to `-`, they will appear BEFORE the content output).
* `--stderr {file path or -}` will emit errors to the given location.
* `--user-agent {value}` will send the given user agent via HTTP instead of the default.
* `--insecure` will ignore invalid HTTPS certificates.
* `--fail` will suppress outputs ONLY ON FAILURE and just return a failure error code (non-zero); success will still output by default. 
* `--silent` will not emit any output regardless of success or failure.
* `--show-error` will show error info even if silent/fail modes on.
* `--include` will include emit returned headers and output to the output path (effectively `-D - -o -`, or `-D file1 -o file1`).
* `--head` will suppress content output and will emit the headers to the "content" output location. This means that `--head -o /tmp/1` is the same as `-D /tmp/1 -o /dev/null`.
* `--user` allows you to specify a `username:password` style authentication header.
* `--referer` specifies the `Referer` HTTP header.
* `--header` (repeatable) allows you to specify any valid HTTP header, and will override defaults set by other parameters (such as `-d` or `--form`).
* `--cookie` (repeatable) allows you to specify an HTTP cookie (as a string, or as a file containing the cookie definition.
* `--cookie-jar` allows you to store HTTP cookies for multiple invocations.

# File/Form/Upload Arguments

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

# Error Codes
- 6: Response present, but a status code >= 400 (e.g. failing) was returned
- 7: No response, but an error was thrown
- 8: Invalid/missing URL
- 9: Unable to read upload file
- 10: Unable to write output file (cookies or output)
- 11: Unable to write to stdout/stderr
- 249: No such host or invalid scheme
- 250: Invalid/missing url

# Command Line 
All command line options *NO LONGER* needs to be specified before the URL - this was a limitation of golang's `flag` module, but I have upgraded to using `spf13/pflag` so this is no longer a problem.

# Tests
Tests are now present in the code - run `go test -v ./...` to run them.

# License
go-curling is licensed under the [MIT License](./LICENSE). Previously it was licensed under the LGPL - as I am the sole author prior to this change, I approve the change.
