FROM golang:1.13-alpine
FROM alpine
RUN apk add --no-cache bash

WORKDIR /app

# Set binary
ARG PROJECT
COPY $PROJECT .

# Set timezone to ShangHai
COPY --from=0 /usr/local/go/lib/time/zoneinfo.zip .
ENV ZONEINFO /app/zoneinfo.zip
RUN apk add --update tzdata \
    && cp /usr/share/zoneinfo/Asia/Ho_Chi_Minh /etc/localtime \
    && echo "Asia/Ho_Chi_Minh" > /etc/timezone \
    && apk del tzdata && rm -rf /var/cache/apk/*

# Set locales
RUN mkdir /app/locales

# Set Musl C lib
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2


## build in container
# FROM golang:alpine AS build-env
# ARG PROJECT
# ARG GitReversion
# ARG BuildTime
# ARG BuildGoVersion
# RUN apk update && apk add upx

# ADD . /src
# RUN cd /src && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s -X main.gitReversion=${GitReversion} -X 'main.buildTime=${BuildTime}' -X 'main.buildGoVersion=${BuildGoVersion}'" -o $PROJECT && upx $PROJECT
