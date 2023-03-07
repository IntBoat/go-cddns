FROM golang:alpine as builder
MAINTAINER Zedo.dev<info@zedo.dev>
WORKDIR /src
ENV TZ="Asia/Hong_Kong"

COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
RUN apk add --no-cache ca-certificates upx git tzdata && update-ca-certificates 2>/dev/null || true
RUN sh ./build.sh

FROM scratch as runner
MAINTAINER Zedo.dev<info@zedo.dev>
ENV TZ="Asia/Hong_Kong"
COPY --from=builder /usr/share/zoneinfo/Asia/Hong_Kong /etc/localtime
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/server /app/go-cddns
CMD ["/app/server"]

WORKDIR /app
CMD ["./go-cddns", "--config=/etc/config.json"]
