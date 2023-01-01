FROM golang:1.19-alpine

WORKDIR /src
COPY go.mod main.go /src/
RUN go build -o /curl .

CMD [ "/curl" ]
