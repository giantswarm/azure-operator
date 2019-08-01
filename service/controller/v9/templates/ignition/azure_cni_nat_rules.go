package ignition

const AzureCNINatRules = `[Unit]
Description=Setup Nat rules for Azure CNI
Wants=systemd-networkd.service
After=systemd-networkd.service
Before=docker.service
[Service]
Type=oneshot
ExecStart=/bin/sh -c "iptables -t nat -A POSTROUTING -m addrtype ! --dst-type local ! -d {{.VnetCIDR}} -j MASQUERADE"
 [Install]
WantedBy=multi-user.target
`
