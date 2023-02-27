FROM golang:1.19-alpine AS build

WORKDIR /src
COPY go.mod go.sum main.go /src/
RUN go build -o /bin/curl .

FROM golang:1.19-alpine AS final
COPY --from=build /bin/curl /bin/curl
ENTRYPOINT [ "/bin/curl" ]
