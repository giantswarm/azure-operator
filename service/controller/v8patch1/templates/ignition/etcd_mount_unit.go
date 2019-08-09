package ignition

const EtcdMountUnit = `[Unit]
Description=Mounts disk to /var/lib/etcd
Before=etcd3.service

[Mount]
What=/dev/disk/by-label/etcd
Where=/var/lib/etcd
Type=ext4

[Install]
WantedBy=multi-user.target
`
