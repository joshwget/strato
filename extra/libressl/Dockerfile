FROM ubuntu
# FROM amd64=ubuntu:16.04 arm64=aarch64/ubuntu:16.04 arm=armhf/ubuntu:16.04
RUN apt-get update && apt-get install -y build-essential wget
RUN wget -P /usr/src/ https://ftp.openbsd.org/pub/OpenBSD/LibreSSL/libressl-2.5.0.tar.gz
RUN cd /usr/src/ && tar xf libressl*
RUN cd /usr/src/libressl* \
    && ./configure \
    --prefix=/usr \
    --sysconfdir=/etc \
    --mandir=/usr/share/man \
    --localstatedir=/var \
    && make

# TODO: remove 1.0.0 first and fix weird permission error
RUN cd /usr/src/libressl* \
    && make install \
    #&& rm /usr/lib/libcrypto.so.1.0.0 \
    && ln -s /usr/lib/libcrypto.so.38.0.1 /usr/lib/libcrypto.so.1.0.0
