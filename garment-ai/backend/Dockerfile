FROM m.daocloud.io/docker.io/library/golang:1.22.1-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct

RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/backend ./cmd/api

FROM m.daocloud.io/gcr.io/distroless/base-debian12:latest

COPY --from=builder /bin/backend /backend

EXPOSE 8080

ENTRYPOINT ["/backend"]