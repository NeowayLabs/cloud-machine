FROM scratch

MAINTAINER Guilherme Santos <guilherme.santos@neoway.com.br>

ADD ./machine-up /opt/config/bin/
ADD ./cluster-up /opt/config/bin/
