FROM --platform=linux/amd64    golang:1.25.7            AS build_amd64
FROM --platform=linux/arm64    golang:1.25.7            AS build_arm64
FROM --platform=linux/ppc64le  golang:1.25.7            AS build_ppc64le
FROM --platform=linux/s390x    golang:1.25.7            AS build_s390x
FROM --platform=linux/386      golang:1.25.7            AS build_386
FROM --platform=linux/arm/v7   golang:1.25.7            AS build_arm
# FROM --platform=linux/arm/v6   golang:1.25.3-alpine3.22 AS build_armel
# FROM --platform=linux/mips64le golang:1.25.3-bookworm   AS build_mips64le
FROM --platform=linux/riscv64  golang:1.25.7            AS build_riscv64
FROM build_${TARGETARCH} AS build

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
    CGO_ENABLED=0 go build -o /bin/curl .

FROM --platform=linux/amd64    alpine:3        AS run_amd64
FROM --platform=linux/arm64    alpine:3        AS run_arm64
FROM --platform=linux/ppc64le  alpine:3        AS run_ppc64le
FROM --platform=linux/s390x    alpine:3        AS run_s390x
FROM --platform=linux/386      alpine:3        AS run_386
FROM --platform=linux/arm/v7   alpine:3        AS run_arm
#FROM --platform=linux/arm/v6   alpine:3        AS run_armel
#FROM --platform=linux/mips64le debian:bookworm AS run_mips64le
FROM --platform=linux/riscv64  alpine:3        AS run_riscv64
FROM run_${TARGETARCH}

COPY --from=build /bin/curl /bin/curl
ENTRYPOINT [ "/bin/curl" ]
