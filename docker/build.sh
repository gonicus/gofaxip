#!/bin/bash

set -e -x

REPO="/usr/src/gofaxip"
BUILD="${HOME}/gofaxip"
ROOT="${HOME}/debroot"
OUTPUT="${HOME}/packages"

rm -rf ${BUILD} ${ROOT} ${OUTPUT}
cp -r ${REPO} ${BUILD}

shortid="$(cd ${REPO} && git rev-parse --short HEAD)"
timestamp="$(date -d @$(cd ${REPO} && git log -1 --pretty=format:%ct) -u +%Y%m%d%H%M)"
version="1.0~git${timestamp}.${shortid}"
debversion="${version}"

cd ${BUILD}
export GOPATH=$(pwd)
go install -ldflags "-X main.version '${version}'" gofaxd gofaxsend

mkdir -p ${ROOT} ${OUTPUT}
mkdir -p ${ROOT}/usr/bin ${ROOT}/etc ${ROOT}/lib/systemd/system ${ROOT}/usr/share/doc/gofaxip
cp ${BUILD}/bin/* ${ROOT}/usr/bin/
cp ${BUILD}/AUTHORS ${BUILD}/LICENSE ${BUILD}/README.md ${ROOT}/usr/share/doc/gofaxip/
cp -r ${BUILD}/config ${ROOT}/usr/share/doc/gofaxip/config
cp ${BUILD}/config/gofax.conf ${ROOT}/etc/
cp ${BUILD}/docker/gofaxip.service ${ROOT}/lib/systemd/system/
cp ${BUILD}/docker/freeswitch.service ${ROOT}/lib/systemd/system/
cp -r ${BUILD}/config/freeswitch ${ROOT}/etc/freeswitch
#ToDo: Chown config

fpm -f -s dir -t deb -n gofaxip \
	-C "${ROOT}" \
	-v "${version}" \
	-p ${OUTPUT}/gofaxip_VERSION_ARCH.deb \
	--description "GOfax.IP - T.38 / Fax Over IP backend for HylaFAX using FreeSWITCH" \
	--category comm \
	--license GPL2 \
	--vendor "GONICUS GmbH" \
	--url "https://github.com/gonicus/gofaxip" \
	--maintainer "Markus Lindenberg <lindenberg@gonicus.de>" \
	--config-files /etc/gofax.conf \
	--config-files /etc/freeswitch \
	-d hylafax-server \
	-d freeswitch \
	-d freeswitch-mod-commands \
	-d freeswitch-mod-dptools \
	-d freeswitch-mod-event-socket \
	-d freeswitch-mod-sofia \
	-d freeswitch-mod-spandsp \
	-d freeswitch-mod-tone-stream \
	-d freeswitch-mod-db \
	-d freeswitch-mod-syslog \
	-d freeswitch-mod-logfile \
	--conflicts freeswitch-sysvinit \
	--conflicts freeswitch-systemd \
	.
