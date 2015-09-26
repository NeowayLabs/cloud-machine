version ?= latest
WORKDIR="github.com/NeowayLabs/cloud-machine"
IMAGENAME=neowaylabs/cloud-machine
IMAGE=$(IMAGENAME):$(version)

all: gobuild
	@echo "Create: machine-up & cluster-up"

gobuild:
	go get -d -v ./...
	go build -v machine-up.go auth.go
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

