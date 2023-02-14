FROM golang:1.19 AS builder

WORKDIR /app

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
RUN go mod download
RUN go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/courier cmd/courier/main.go

FROM scratch

COPY --from=builder /go/bin/courier /go/bin/courier

EXPOSE 8080
EXPOSE 8081

ENTRYPOINT ["/go/bin/courier"]
