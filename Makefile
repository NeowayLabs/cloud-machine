version ?= 1.0
WORKDIR="github.com/NeowayLabs/cloud-machine"
IMAGENAME=neowaylabs/cloud-machine
IMAGE=$(IMAGENAME):$(version)

all: machine-up cluster-up
	@echo "Created: machine-up & cluster-up"

goget:
	go get -d -v ./...

machine-up:
	cd cmd/machine-up && make build

cluster-up:
	cd cmd/cluster-up && make build

build: machine-up cluster-up

static:
	cd cmd/machine-up && make static
	cd cmd/cluster-up && make static
	ldd cmd/machine-up/machine-up | grep "not a dynamic executable"
	ldd cmd/cluster-up/cluster-up | grep "not a dynamic executable"

publish: image
	docker push $(IMAGE)

image: build
	docker build -t $(IMAGE) .

build-static: build-env
	docker run --rm -v `pwd`:/go/src/$(WORKDIR) --privileged -i -t $(IMAGENAME) make static

build-env:
	docker build -t $(IMAGENAME) -f ./hack/Dockerfile .

shell: build-env
	docker run --rm -v `pwd`:/go/src/$(WORKDIR) --privileged -i -t $(IMAGENAME) bash
