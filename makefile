#
# 如果版本发生变化需要修改这里的版本号，或者在命令行指定版本号，如: make VER=1.0.0110 deploy
# build.py中的版本号可以通过参数指定，无需改动
# 如果要在生产环境部署应用可以指定ENV参数，如: make ENV=prod deploy
#
APP    := taskd
VER    := 1.0.250604
OS     := $(shell go env GOOS)
ARCH   := $(shell go env GOARCH)
ENV    := test
SHENMA_DOCKER_REPO := $(shell grep '^SHENMA_DOCKER_REPO=' ./.env | cut -d '=' -f 2-)
SHENMA_DOCKER_HOST := $(shell grep '^SHENMA_DOCKER_HOST=' ./.env | cut -d '=' -f 2-)
SHENMA_DOCKER_USER := $(shell grep '^SHENMA_DOCKER_USER=' ./.env | cut -d '=' -f 2-)
SHENMA_DOCKER_PASSWORD := $(shell grep '^SHENMA_DOCKER_PASSWORD=' ./.env | cut -d '=' -f 2-)
UPLOAD_GITHUB := "../upload-docker-images"

#ENV := prod
EXEEXT ?= 
ifeq (windows,$(OS))
EXEEXT := .exe
endif

ifdef DEBUG
DEBUGOPT := '--debug'
else
DEBUGOPT := 
endif
# 构建
build:
	python ./build.py --software $(VER) $(DEBUGOPT)

docs:
	swag init

# 打镜像包
package: 
	docker build -t $(APP):$(VER) .

# 上传镜像包到dockerhub
upload_dockerhub:
	docker tag $(APP):$(VER) $(SHENMA_DOCKER_REPO)/$(APP):$(VER)
	docker login $(SHENMA_DOCKER_HOST) -u $(SHENMA_DOCKER_USER) -p $(SHENMA_DOCKER_PASSWORD)
	docker push $(SHENMA_DOCKER_REPO)/$(APP):$(VER)

upload_github:
	docker tag $(APP):$(VER) zgsm/$(APP):$(VER)
	docker save -o $(APP)-$(VER).tar zgsm/$(APP):$(VER)
	mv $(APP)-$(VER).tar $(UPLOAD_GITHUB)/images/
	@echo -------TODO: upload image to dockerhub-------
	@echo cd $(UPLOAD_GITHUB)
	@echo git add images/$(APP)-$(VER).tar
	@echo git commit -am \"upload zgsm/$(APP):$(VER)\"
	@echo git push origin main

# 上传镜像包到制品库和前置harbor
upload: upload_dockerhub

DEPLOY_YAML := "./__$(APP)_$(ENV)_$(VER).yaml"
# 生成服务部署的YAML配置
genyaml:
	echo generate $(DEPLOY_YAML) ...
	bash shenma-secret.sh -d $(APP) -p $(ENV) -v $(VER) -t ./$(APP).template.yaml

apply:
	kubectl delete -f $(DEPLOY_YAML)
	kubectl apply -f $(DEPLOY_YAML)

k8s_clean:
	kubectl delete -f $(DEPLOY_YAML)

k8s_create:
	kubectl apply -f $(DEPLOY_YAML)

# 部署
deploy: package upload genyaml apply

test:
	@for script in `ls test/*.sh`; do				\
		echo sh ./$${script};						\
		sh ./$${script} || exit $?;					\
	done

.PHONY: docs build package upload deploy upload_dockerhub test genyaml apply k8s_clean k8s_create docker
