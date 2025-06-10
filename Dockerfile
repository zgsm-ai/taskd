FROM golang:1.24.0 AS builder
WORKDIR /app
COPY . .

RUN go env -w CGO_ENABLED=0 && \
    go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct  

#
#go env -w GOPROXY=http://mirrors.sangfor.org/nexus/repository/go-proxy-group
#

RUN go mod tidy && go build -o taskd *.go

FROM centos:7.6.1810
#时区设置
ENV env prod
ENV TZ Asia/Shanghai
WORKDIR /
# RUN curl -O http://10.72.1.16:30574/tools/kubectl
# RUN chmod 777 kubectl
# RUN mv kubectl /usr/bin/
# RUN curl -O http://10.72.1.16:30574/tools/jq
# RUN chmod 777 jq
# RUN mv jq /usr/bin/
COPY --from=builder /app/taskd /usr/local/bin
RUN chmod 755 /usr/local/bin/taskd
ENTRYPOINT ["/usr/local/bin/taskd"]

