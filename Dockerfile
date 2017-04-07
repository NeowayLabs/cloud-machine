FROM scratch

MAINTAINER Guilherme Santos <guilherme.santos@neoway.com.br>

ADD ./cmd/machine-up/machine-up /opt/cloud-machine/bin/
ADD ./cmd/cluster-up/cluster-up /opt/cloud-machine/bin/
