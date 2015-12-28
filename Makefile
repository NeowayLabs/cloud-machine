version ?= latest
WORKDIR="github.com/NeowayLabs/cloud-machine"
IMAGENAME=neowaylabs/cloud-machine
IMAGE=$(IMAGENAME):$(version)

all: machine-up cluster-up
	@echo "Create: machine-up & cluster-up"

goget:
	go get -d -v ./...

machine-up: goget
	go build -v machine-up.go auth.go

cluster-up: goget
	go build -v cluster-up.go auth.go

install: deploy

publish: image
	docker push $(IMAGE)

image: build
	docker build -t $(IMAGE) .

build: build-env
	docker run --rm -v `pwd`:/go/src/$(WORKDIR) --privileged -i -t $(IMAGENAME) bash hack/make.sh

build-env:
	docker build -t $(IMAGENAME) -f ./hack/Dockerfile .

check: build
	docker run --rm -v `pwd`:/go/src/$(WORKDIR) --privileged -i -t $(IMAGENAME) bash hack/check.sh

shell: build
	docker run --rm -v `pwd`:/go/src/$(WORKDIR) --privileged -i -t $(IMAGENAME) bash

