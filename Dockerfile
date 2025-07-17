FROM golang:1.24.0 AS builder
WORKDIR /app
COPY . .

RUN go env -w CGO_ENABLED=0 && \
    go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy,direct

RUN go mod tidy
RUN go build -ldflags="-s -w" -o taskd *.go
RUN chmod 755 taskd

FROM alpine:3.21 AS runtime

#时区设置
ENV env prod
ENV TZ Asia/Shanghai
WORKDIR /

COPY --from=builder /app/taskd /usr/local/bin/taskd
ENTRYPOINT ["/usr/local/bin/taskd"]

