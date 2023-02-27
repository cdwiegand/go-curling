# go-curling
This program was designed to replace the curl that is no longer shipped with Microsoft's dotnet core docker containers. Removing that kept breaking all of my upgraded containers, and I really wanted curl back for healthchecks without having to `apt install` and `apt clean` and cleaning out the cache. So I built a simple curl that handled the healthcheck calls I was doing.

# Command Line 
All command line options *NO LONGER* needs to be specified before the URL - this was a limitation of golang's `flag` module, but I have upgraded to using `spf13/pflag` so this is no longer a problem.

# Arguments
| short | long form | default | description |
| -- | -- | -- | -- |
| `-X` | `--method` | `GET` | HTTP method to use |
| `-o` | `--output` | `-` | Where to output results |
| `-D` | `--dump-header` | `/dev/null` | Where to output headers |
| `-A` | `--user-agent` | `go-curling/1` | User-agent to use |
| `-f` | `--silent` | `false` | If fail do not emit contents just return fail exit code (-6) |
| `-k` | `--insecure` | `false` | Ignore invalid SSL certificates |

# Examples

```
curl -D - -o - https://google.com
curl -D /dev/null -o /dev/null https://google.not.valid.haha
curl https://google.com
curl https://my.local.test:443 -k
```

# Using in a Dockerfile
```
COPY --from=ghcr.io/cdwiegand/cdwiegand/go-curling:latest /curl /usr/bin/curl
HEALTHCHECK CMD curl -A "HealthCheck-Docker/1.0" http://localhost:80
```

Alternative:
```
COPY --from=cdwiegand/go-curling:latest /curl /usr/bin/curl
HEALTHCHECK CMD curl -A "HealthCheck-Docker/1.0" http://localhost:80
```