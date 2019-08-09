package ignition

const AzureCNINatRules = `[Unit]
Description=Setup iptables Nat rules for Azure CNI
Wants=systemd-networkd.service
After=systemd-networkd.service
Before=docker.service
[Service]
Type=oneshot
ExecStartPre=/bin/sh -c "iptables -I INPUT 1 -m udp -p udp --source-port 80 -j DROP"
ExecStartPre=/bin/sh -c "iptables -I INPUT 1 -m udp -p udp --source-port 443 -j DROP"
ExecStart=/bin/sh -c "iptables -t nat -A POSTROUTING -m addrtype ! --dst-type local ! -d {{.VnetCIDR}} -j MASQUERADE"
[Install]
WantedBy=multi-user.target
`
