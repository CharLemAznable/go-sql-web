##
# First Build
#
##
FROM golang:1.11.1-alpine AS builder
ENV GO111MODULE on
COPY . /go/src/github.com/bingoohuang/go-sql-web/
WORKDIR /go/src/github.com/bingoohuang/go-sql-web/

RUN set -x \
	echo 'http://mirrors.aliyun.com/alpine/v3.8/community/' > /etc/apk/repositories \
	&& echo 'http://mirrors.aliyun.com/alpine/v3.8/main/' >> /etc/apk/repositories \
	&& apk add --no-cache --virtual .build-deps git build-base curl \
	&& git config --global http.sslVerify true \
	&& GO111MODULE=off go get -u -v github.com/jteeuwen/go-bindata/... \
	&& GO111MODULE=off go get -u -v golang.org/x/tools/cmd/goimports \
	&& sh download-static.sh \
	&& go-bindata -ignore=res/.DS_Store res/... \
	&& goimports -w bindata.go \
	&& go mod tidy \
	&& GOOS=linux GOARCH=amd64 go build -a -o app . \
	&& apk del .build-deps
##
# Second
#
##
FROM alpine:3.7
RUN apk --no-cache add ca-certificates tzdata  && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /root/
COPY --from=builder /go/src/github.com/bingoohuang/go-sql-web/app .
COPY --from=builder /go/src/github.com/bingoohuang/go-sql-web/appConfig.toml .
COPY --from=builder /go/src/github.com/bingoohuang/go-sql-web/static ./static/



EXPOSE 8381
ENTRYPOINT ["./app"]


