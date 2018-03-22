#!/bin/bash

set -e
cp -r /input /build/source

cd /build/source
git checkout $VERSION || exit 1

if [ "$VERSION" == "master" ]; then
    snapshot_version=0:$(git log --format="%cd" --date=format:'%Y-%m-%d-%H-%M-%S' -n 1)-$(git log --format="%H" -n 1)
    dch -b -v $snapshot_version build from git
fi

chown -R build: /build/source
cd /tmp && mk-build-deps -i -r /build/source/debian/control
cd /build/source && sudo -u build debuild -us -uc -b

OWNER=$(stat -c %u:%g /output) && \
    find /build -maxdepth 1 -type f \
    -execdir cp -f -t /output '{}' \; \
    -execdir chown "$OWNER" '/output/{}' \; 
