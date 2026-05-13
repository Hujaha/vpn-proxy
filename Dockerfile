FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/vpn-proxy .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/vpn-proxy /app/vpn-proxy
ENV VPN_PROXY_ADDR=:2053 \
    VPN_PROXY_DB=/data/vpn-proxy.db
EXPOSE 2053
VOLUME ["/data"]
ENTRYPOINT ["/app/vpn-proxy"]
