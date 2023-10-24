.PHONY:

# build go
PROJECT_NAME = ${shell git config --local remote.origin.url | xargs basename | sed  's/.git//'}
BuildGoVersion = $(shell go version)
GitReversion = $(shell git rev-parse HEAD)
BuildTime = $(shell date +'%Y.%m.%d.%H%M%S')
Username = $(shell git show | grep 'Author' |sed -n 1p |awk '{print $2}')
# internal etcd 
# ETCD = "172.31.11.213:2379,172.31.9.82:2379,172.31.3.115:2379"
# external etcd 
ETCD = "43.198.80.174:2379,16.163.142.133:2379,16.163.139.84:2379"
# internal json
# JSON = /vn/p3/p3.toml
# external json
JSON = /vn/p3/p3-test.toml
SOCK5 = ''

# build image
IMAGE_REGISTRY = p3-harbor.168-system.com
IMAGE_PATH = p3/backend/p3/
BRANCH = $(shell git branch --show-current)
SHA = $(shell git rev-parse --short HEAD)



run:  
	go build -ldflags "-w -s  -X 'main.buildTime=${BuildTime}' -X 'main.username=${Username}' -X main.gitReversion=${GitReversion} -X 'main.buildGoVersion=${BuildGoVersion}'" -o ${PROJECT_NAME}
	./${PROJECT_NAME} ${ETCD} ${JSON} ${SOCK5} 
 	

init:
	git submodule init
	git submodule update --remote
	go mod tidy

# run-docker:  build-image
# 	docker rm -f $(LOCAL_CONTAINER_NAME)
# 	docker run --name  $(LOCAL_CONTAINER_NAME) -itd -p $(SERVICE_PORT):$(SERVICE_PORT) $(TARGET_IMAGE)/$(BRANCH):$(SHA)

# image-push:  build-image
# 	docker push $(TARGET_IMAGE)/$(BRANCH):$(SHA)
# 	docker tag $(TARGET_IMAGE)/$(BRANCH):$(SHA) $(TARGET_IMAGE)/$(BRANCH):latest
# 	docker push $(TARGET_IMAGE)/$(BRANCH):latest

 
# image-push-dirty:  
# 	docker push $(TARGET_IMAGE)/$(BRANCH):$(SHA)
# 	docker tag $(TARGET_IMAGE)/$(BRANCH):$(SHA) $(TARGET_IMAGE)/$(BRANCH):latest
# 	docker push $(TARGET_IMAGE)/$(BRANCH):latest

# deploy-k8s:
# 	kubectl config use-context $(KUBE_CONFIG_NAME)
# 	kubectl set image deployment/$(KUBE_POD_NAME) $(KUBE_POD_NAME)=$(TARGET_IMAGE)/$(BRANCH):$(SHA)  -n $(KUBE_NAMESPACE)

# ci: image-push deploy-k8s
	
