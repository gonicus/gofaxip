#!/bin/bash

# script to workaround the issues caused by https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=1076953

echo "ensure hylafax packages are on hold"
apt-mark hold hylafax-server
apt-mark hold hylafax-client

echo "ensure mount/umount statements in /usr/sbin/hylafax_wrapper are inactive"
sed -i 's/\(^[^#]*mount*\)/#\1/g' /usr/sbin/hylafax_wrapper


echo "ensure permanent bindmount"
cat << EOF > /etc/systemd/system/var-spool-hylafax-etc.mount
[Mount]
What=/etc/hylafax
Where=/var/spool/hylafax/etc
Type=none
Options=bind

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable var-spool-hylafax-etc.mount
systemctl start var-spool-hylafax-etc.mount

