package ignition

const EphemeralDiskMountPointFixUnit = `[Unit]
Description=Changes ephemeral disk mount point to /var/lib
Before=waagent.service

[Service]
Type=oneshot
ExecStart=sed -i'' -e 's|ResourceDisk.MountPoint=/mnt/resource|ResourceDisk.MountPoint=/var/lib|' /usr/share/oem/waagent.conf

[Install]
WantedBy=multi-user.target
`
