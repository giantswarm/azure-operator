package ignition

const KubeletMountUnit = `[Unit]
Description=Mounts disk to /var/lib/kubelet
Before=k8s-kubelet.service

[Mount]
What=/dev/disk/by-label/kubelet
Where=/var/lib/kubelet
Type=xfs

[Install]
WantedBy=multi-user.target
`
