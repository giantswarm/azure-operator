package ignition

const VNICConfigurationUnit = `[Unit]
Description=VNIC configuration
Wants=systemd-networkd.service
After=systemd-networkd.service
Before=docker.service
[Service]
Type=oneshot
ExecStart=/bin/sh -c "ethtool -G eth0 tx 1024"
 [Install]
WantedBy=multi-user.target
`
