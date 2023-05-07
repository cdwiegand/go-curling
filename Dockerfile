FROM golang:alpine AS build

WORKDIR /src
COPY go.mod go.sum main.go /src/
RUN go build -o /bin/curl .

FROM golang:alpine AS final
COPY --from=build /bin/curl /bin/curl
ENTRYPOINT [ "/bin/curl" ]
