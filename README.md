# go-curling
This program was designed to replace the curl that is no longer shipped with Microsoft's dotnet core docker containers. Removing that kept breaking all of my upgraded containers, and I really wanted curl back for healthchecks without having to `apt install` and `apt clean` and cleaning out the cache. So I built a simple curl that handled the healthcheck calls I was doing.

# Command Line 
All command line options *MUST* be specified before the URL - this is a limitation of golang's `flag` module.

# Examples

```
curl -D - -o - https://google.com
curl -D /dev/null -o /dev/null https://google.not.valid.haha
curl https://google.com
```

# Using in a Dockerfile
```
COPY --from=ghcr.io/cdwiegand/cdwiegand/go-curling:latest /curl /usr/bin/curl
HEALTHCHECK CMD curl -A "HealthCheck: Docker/1.0" http://localhost:80
```