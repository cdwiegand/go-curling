# go-curling
This program was designed to replace the curl that is no longer shipped with Microsoft's dotnet core docker containers. Removing that kept breaking all of my upgraded containers, and I really wanted curl back for healthchecks without having to `apt install` and `apt clean` and cleaning out the cache. So I built a simple curl that handled the healthcheck calls I was doing.

# Command Line 
All command line options are ignored unless they are a HTTP/s url (that is, if they start with `http://` or `https://`). Error code 0 will be returned if all URL(s) present return a successful response (<400). Redirects are followed.