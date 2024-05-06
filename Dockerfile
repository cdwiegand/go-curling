FROM golang:alpine AS build

LABEL org.opencontainers.image.authors="Chris Wiegand"
LABEL org.opencontainers.image.source="https://github.com/cdwiegand/go-curling"
LABEL org.opencontainers.image.documentation="https://github.com/cdwiegand/go-curling/README.md"
LABEL org.opencontainers.image.base.name="ghcr.io/cdwiegand/go-curling:latest"
LABEL org.opencontainers.image.description="Reimplementation of curl in golang"
LABEL org.opencontainers.image.licenses="LGPL-2.1-or-later"
LABEL org.opencontainers.image.title="go-curling"

WORKDIR /src
COPY go.mod go.sum main.go /src/
RUN sed -i "s/##DEV##/(`date -Idate`)/" /src/main.go && go build -o /bin/curl .

FROM golang:alpine AS final
COPY --from=build /bin/curl /bin/curl
ENTRYPOINT [ "/bin/curl" ]
