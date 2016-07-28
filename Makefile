version ?= latest
WORKDIR="github.com/NeowayLabs/cloud-machine"
IMAGENAME=neowaylabs/cloud-machine
IMAGE=$(IMAGENAME):$(version)

all: build install
	@echo "Created: machine-up & cluster-up"

goget:
	go get -d -v ./...

machine-up:
	cd cmd/machine-up && make build

cluster-up:
	cd cmd/cluster-up && make build

build: goget machine-up cluster-up

install: build
	cd cmd/machine-up && make install
	cd cmd/cluster-up && make install

build-static:
	cd cmd/machine-up && make build-static
	cd cmd/cluster-up && make build-static
	ldd cmd/machine-up/machine-up | grep "not a dynamic executable"
	ldd cmd/cluster-up/cluster-up | grep "not a dynamic executable"

publish: build-image
	docker push $(IMAGE)

build-image: static
	docker build -t $(IMAGE) .

build-docker: build-dev-image
	docker run --rm -v `pwd`:/go/src/$(WORKDIR) --privileged -i -t $(IMAGENAME) make build

build-docker-static: build-dev-image
	docker run --rm -v `pwd`:/go/src/$(WORKDIR) --privileged -i -t $(IMAGENAME) make build-static

build-dev-image:
	docker build -t $(IMAGENAME) -f ./hack/Dockerfile .

shell: build-dev-image
	docker run --rm -v `pwd`:/go/src/$(WORKDIR) --privileged -i -t $(IMAGENAME) bash
