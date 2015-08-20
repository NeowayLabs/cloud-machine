NAME="neowaylabs/cloud-machine"
REGISTRY="hub.docker.com"
IMAGE=$(REGISTRY)/$(NAME):1.0.0

all: image
	@echo "Create image: ${IMAGE}"

install: deploy

image: build
	docker build -t $(IMAGE) .

deploy: image
	docker push $(IMAGE)

build: build-env
	docker run --rm -v `pwd`:/go/src/$(NAME) --privileged -i -t $(NAME) bash hack/make.sh

build-env:
	docker build -t $(NAME) -f ./hack/Dockerfile .

check: build
	docker run --rm -v `pwd`:/go/src/$(NAME) --privileged -i -t $(NAME) bash hack/check.sh

shell: build
	docker run --rm -v `pwd`:/go/src/$(NAME) --privileged -i -t $(NAME) bash

