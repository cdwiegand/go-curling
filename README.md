# Purpose
This program was designed to replace the curl that is no longer shipped with Microsoft's dotnet core docker containers. Removing that kept breaking all of my upgraded containers, and I really wanted curl back for healthchecks without having to `apt install` and `apt clean` and cleaning out the cache. So I built a simple curl that handled the healthcheck calls I was doing. I have since started expanding it to meet more needs of the [original curl](https://curl.se/), while remaining golang based.

# Arguments
| short | long form | default | type | description |
| -- | -- | -- | -- | -- |
| `-V` | `--version` | (none) | (none) | Return version and exit |
| `-X` | `--method` | `GET` | string | HTTP method to use (generally `GET` unless using `-I` or similar parameters) |
| `-o` | `--output` | `-` (/dev/stdout) | `-` or file-path | Where to output results |
| `-D` | `--dump-header` | `/dev/null` | `-` or file-path | Where to output headers separately |
|  | `--stderr` | `/dev/stderr` | `-` or file-path | Log errors to this replacement for stderr |
| `-A` | `--user-agent` | `go-curling/1` | string | User-agent to use |
| `-k` | `--insecure` | `false` | boolean | Ignore invalid SSL certificates |
| `-f` | `--fail` | `false` | boolean | If fail do not emit contents just return fail exit code (-6) |
| `-s` | `--silent` | `false` | boolean | Do not emit any output (unless overridden with `show-error`) |
| `-S` | `--show-error` | `false` | boolean | Show error info even if silent/fail modes on |
| `-i` | `--include` | `false` | boolean | Prepend returned headers to body output |
| `-I` | `--head` | `false` | boolean | Only emit headers returned, ignore body |
| `-u` | `--user` | (none) | string | Username:Password for HTTP Basic Authentication |
| `-e` | `--referer` | (none) | URI | HTTP referer header |
| `-b` | `--cookie` | (none) | HTTP cookie string or `@`file-path | Specifies cookie header (if `=` present) or file from which to read cookies from, read-only |
| `-c` | `--cookie-jar` | (none) | file-path | Specifies file to which to write cookies to |
| `-d` | `--data` | (none) | name=value OR name=`@`file-path | Send next parameter as POST / `application/x-www-form-urlencoded` |
| `-F` | `--form` | (none) | name=value OR name=`@`file-path |Send next parameter as POST / `multipart/form-data` |
| `-T` | `--upload-file` | (none) | file-path | File to upload (PUT method by default) |

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
HEALTHCHECK CMD curl -f http://localhost:80
```

# Needs
- Needs automated tests, esp. against a known HTTP server that can return explicit info like our referer, basic auth info, etc.. echoing back for testing purposes.

# Error Codes
- 6: Response present, but a status code >= 400 (e.g. failing) was returned
- 7: No response, but an error was thrown
- 8: Invalid/missing URL
- 9: Unable to read upload file

# Command Line 
All command line options *NO LONGER* needs to be specified before the URL - this was a limitation of golang's `flag` module, but I have upgraded to using `spf13/pflag` so this is no longer a problem.

# Tests (manual for now)
```
go run ./main.go https://httpbin.org/get?test=one

go run ./main.go -d test=one https://httpbin.org/post
go run ./main.go -d test=@raw.data https://httpbin.org/post
go run ./main.go -d @form.data https://httpbin.org/post

go run ./main.go -F test=one https://httpbin.org/post
go run ./main.go -F test=@raw.data https://httpbin.org/post
go run ./main.go -F @form.data https://httpbin.org/post

go run ./main.go -T raw.data https://httpbin.org/post

go run ./main.go https://httpbin.org/delete -X DELETE
```