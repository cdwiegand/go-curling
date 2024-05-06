# Purpose
This program was designed to replace the curl that is no longer shipped with Microsoft's dotnet core docker containers. Removing that kept breaking all of my upgraded containers, and I really wanted curl back for healthchecks without having to `apt install` and `apt clean` and cleaning out the cache. So I built a simple curl that handled the healthcheck calls I was doing. I have since started expanding it to meet more needs of the [original curl](https://curl.se/), while remaining golang based.

# Differences between original curl and go-curling
Not all functionality is supported, but one difference that makes this not 100% drop-in would be that the `--cookie-jar`/`-c` is both read and write - the `--cookie` / `-b` command only specifies a raw HTTP cookie on the command line - it is not usable as a file to read a prior cookie jar, due to the custom JSON format for storing cookies. So normally if you want to use cookies to login a session, just use `--cookie-jar`/`-c` in each all - no need to specify `--cookie`/`-b` unless you want to specify a "starting" cookie value.

- Globbing is NOT supported
- Environment variable interpolation ("Variables" in the curl man page) is not supported
- Command line arguments not listed below are also not supported
- go-curling does not implement global vs scoped arguments - `-:` / `--next` is not supported

Note that one thing that is now supported is that if you specify multiple URLs, you can specify multiple `-o` or `-D` values and go-curling will honor that, but if you specify more URLs than you have specified outputs, the extra URLs will be processed with the default value for the given flag (content output to stdout).

# Arguments
| short | long form | default | type | description |
| -- | -- | -- | -- | -- |
| `-V` | `--version` | (none) | (none) | Return version and exit |
| `-X` | `--method` | `GET` | string | HTTP method to use (generally `GET` unless using `-I` or similar parameters) |
| `-o` | `--output` | `-` (/dev/stdout) | `-` or file-path(s) | Where to output results |
| `-D` | `--dump-header` | `/dev/null` | `-` or file-path(s) | Where to output headers separately |
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
| `-T` | `--upload-file` | (none) | file-path | File(s) to upload to given URL(s) (PUT method by default) |

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
Tests are now present in the code - run `go test` to run them.

# License
go-curling is [licensed](./LICENSE) under the [LGPL 2.1 or later](./COPYRIGHT)
