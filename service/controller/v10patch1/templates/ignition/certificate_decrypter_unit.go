package ignition

const CertificateDecrypterUnit = `[Unit]
Description=Certificate Decrypter
Wants=k8s-setup-network-env.service
After=k8s-setup-network-env.service
Before=k8s-kubelet.service etcd3.service
[Service]
Type=oneshot
EnvironmentFile=/etc/.enc/key
EnvironmentFile=/etc/.enc/iv
ExecStart=/bin/sh -c "\
{{ range $index, $file := .CertsPaths -}}
openssl enc -aes-256-cfb -d -K ${ENCRYPTION_KEY} -iv ${INITIAL_VECTOR} -in {{ $file }}.enc -out {{ $file }} ; \
{{ end -}}
"
 [Install]
WantedBy=multi-user.target
`
