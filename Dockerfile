FROM scratch

MAINTAINER Guilherme Santos <guilherme.santos@neoway.com.br>

ADD ./machine-up /opt/cloud-machine/bin/
ADD ./cluster-up /opt/cloud-machine/bin/
