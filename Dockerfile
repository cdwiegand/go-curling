ARG SOURCE_TAG=1.23
# if linux/riscv64 then SOURCE_TAG=1.23-alpine
FROM golang:${SOURCE_TAG} AS build

LABEL org.opencontainers.image.authors="Chris Wiegand"
LABEL org.opencontainers.image.source="https://github.com/cdwiegand/go-curling"
LABEL org.opencontainers.image.documentation="https://github.com/cdwiegand/go-curling/README.md"
LABEL org.opencontainers.image.base.name="ghcr.io/cdwiegand/go-curling:latest"
LABEL org.opencontainers.image.description="Reimplementation of curl in golang"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.title="go-curling"

WORKDIR /src
COPY . /src
RUN sed -i "s/##DEV##/`date -Idate`/" /src/main.go /src/cli/flags.go && \
    go build -o /bin/curl .

FROM golang:${SOURCE_TAG} AS final
COPY --from=build /bin/curl /bin/curl
ENTRYPOINT [ "/bin/curl" ]
