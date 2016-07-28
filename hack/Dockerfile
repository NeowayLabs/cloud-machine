FROM ubuntu:14.04
MAINTAINER Guilherme Santos <guilherme.santos@neoway.com.br>

# Packaged dependencies
RUN apt-get update && apt-get install -y --force-yes \
    ca-certificates \
	build-essential \
	curl \
	git \
	mercurial \
	--no-install-recommends

# Install Go
ENV GO_VERSION 1.4.2
RUN curl -sSL https://golang.org/dl/go${GO_VERSION}.src.tar.gz | tar -v -C /usr/local -xz \
	&& mkdir -p /go/bin
ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go
RUN cd /usr/local/go/src && ./make.bash --no-clean 2>&1

WORKDIR /go/src/github.com/NeowayLabs/cloud-machine

ENV CGO_ENABLED 0

# Upload docker source
COPY . /go/src/github.com/NeowayLabs/cloud-machine

RUN go get -d
