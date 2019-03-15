package ignition

const DockerMountUnit = `[Unit]
Description=Mounts disk to /var/lib/docker
Requires=format-docker-disk.service
After=format-docker-disk.service
Before=docker.service

[Mount]
What=/dev/disk/by-label/docker
Where=/var/lib/docker
Type=xfs

[Install]
WantedBy=multi-user.target
`
