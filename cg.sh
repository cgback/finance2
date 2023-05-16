#! /bin/bash

git reset --hard
git checkout main
git pull origin main
git submodule init
git submodule update

PROJECT="finance2"
GitReversion=`git rev-parse HEAD`
BuildTime=`date +'%Y.%m.%d.%H%M%S'`
BuildGoVersion=`go version`
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.gitReversion=${GitReversion}  -X 'main.buildTime=${BuildTime}' -X 'main.buildGoVersion=${BuildGoVersion}'" -o $PROJECT
upx $PROJECT

#外网服务器
scp -i /opt/data/superuser -P 10087 $PROJECT superuser@34.92.240.177:/home/centos/workspace/cg/$PROJECT/${PROJECT}_cg
ssh -i /opt/data/superuser -p 10087 superuser@34.92.240.177 "sh /home/centos/workspace/cg/${PROJECT}/cg.sh"
