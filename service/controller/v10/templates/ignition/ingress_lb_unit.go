package ignition

const IngressLBUnit = `[Unit]
Description=Add Kubernetes load balancer manifest to addons
ConditionPathExists=!/srv/k8s-ingress-loadbalancer.yaml.lock
[Service]
Type=oneshot
ExecStart=/bin/sh -c "sed -i '/MANIFESTS=\"\"/a MANIFESTS=\"${MANIFESTS} k8s-ingress-loadbalancer.yaml\"' /opt/k8s-addons && touch /srv/k8s-ingress-loadbalancer.yaml.lock"
[Install]
WantedBy=multi-user.target
`
