FROM arm32v7/ubuntu

RUN apt-get update

RUN apt-get install -y wget openvpn unzip 
RUN apt-get install -y --fix-missing sudo

RUN mkdir -p /etc/openvpn

RUN cd /etc/openvpn && wget https://account.surfshark.com/api/v1/server/configurations && unzip configurations && rm configurations

COPY auth.txt /etc/openvpn

WORKDIR /etc/openvpn


