# go-curling
This program was designed to replace the curl that is no longer shipped with Microsoft's dotnet core docker containers. Removing that kept breaking all of my upgraded containers, and I really wanted curl back for healthchecks without having to `apt install` and `apt clean` and cleaning out the cache. So I built a simple curl that handled the healthcheck calls I was doing.

# Command Line 
All command line options *NO LONGER* needs to be specified before the URL - this was a limitation of golang's `flag` module, but I have upgraded to using `spf13/pflag` so this is no longer a problem.

# Arguments
| short | long form | default | description |
| -- | -- | -- | -- |
| `-X` | `--method` | `GET` | HTTP method to use |
| `-o` | `--output` | `-` | Where to output results |
| `-D` | `--dump-header` | `/dev/null` | Where to output headers separately |
| `-A` | `--user-agent` | `go-curling/1` | User-agent to use |
| `-k` | `--insecure` | `false` | Ignore invalid SSL certificates |
| `-f` | `--fail` | `false` | If fail do not emit contents just return fail exit code (-6) |
| `-s` | `--silent` | `false` | Do not emit any output (unless overridden with `show-error`) |
| `-S` | `--show-error` | `false` | Show error info even if silent/fail modes on |
| `-i` | `--include` | `false` | Prepend headers returned to body output |
| `-I` | `--head` | `false` | Only emit headers returned, ignore body |
| `-u` | `--user` |  | Username:Password for HTTP Basic Authentication |
| `-e` | `--referer` |  | HTTP referer header |
|  | `--stderr` | `/dev/stderr` | Log errors to this replacement for stderr |

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
