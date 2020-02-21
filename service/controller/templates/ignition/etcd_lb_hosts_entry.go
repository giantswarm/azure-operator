package ignition

const EtcdLBHostsEntry = `[Unit]
Description=Adds hosts file entry for etcd LB DNS name that points to loopback IP
After=network-online.target
Wants=network-online.target
[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/sh -c '\
  grep {{.Cluster.Etcd.Domain}} /etc/hosts || echo "127.0.0.1    {{.Cluster.Etcd.Domain}}" >> /etc/hosts'
[Install]
WantedBy=multi-user.target
`
