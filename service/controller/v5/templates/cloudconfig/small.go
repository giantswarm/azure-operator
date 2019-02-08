package cloudconfig

const Small = `---
ignition:
  version: 2.2.0
  config:
    append:
      source: {{ .BlobURL }}
 storage:
  files:
    - path: /etc/systemd/system/waagent.service
      filesystem: root
      mode: 292
      contents:
        source: "oem:///units/waagent.service"
    - path: /etc/.enc/key
      filesystem: root
      mode: 0400
      contents:
        source: "data:text/plain;base64,ENCRYPTION_KEY={{ .EncryptionKey }}"
    - path: /etc/.enc/iv
      filesystem: root
      mode: 0400
      contents:
        source: "data:text/plain;base64,INITIAL_VECTOR={{ .InitialVector }}"
systemd:
  units:
    - name: waagent.service
      enabled: true
`
