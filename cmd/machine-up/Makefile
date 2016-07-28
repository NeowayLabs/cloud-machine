all: build install

build:
	go build

static:
	CGO_ENABLED=0 go build -v -a -installsuffix cgo

install:
	go install
