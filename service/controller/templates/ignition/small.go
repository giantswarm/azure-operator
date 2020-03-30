package ignition

const Small = `{
  "ignition": {
    "version": "2.2.0",
    "config": {
      "append": [
	    {
          "source": "{{ .BlobURL }}"
        }
	  ]
    }
  },
  "storage": {
    "files": [
      {
        "path": "/etc/.enc/key",
        "filesystem": "root",
        "mode": 256,
        "contents": {
          "source": "data:text/plain,ENCRYPTION_KEY={{ .EncryptionKey }}"
        }
      },
      {
        "path": "/etc/.enc/iv",
        "filesystem": "root",
        "mode": 256,
        "contents": {
          "source": "data:text/plain,INITIAL_VECTOR={{ .InitialVector }}"
        }
      }
    ],
  "filesystems": [
      { 
        "name": "docker",
        "mount": {
          "device": "/dev/disk/azure/scsi1/{{ if eq .InstanceRole "master"}}lun1{{ else }}lun21{{end}}",
          "wipeFilesystem": true,
          "label": "docker",
          "format": "xfs"
        }
      },
      { 
        "name": "kubelet",
        "mount": {
          "device": "/dev/disk/azure/scsi1/{{ if eq .InstanceRole "master"}}lun2{{ else }}lun22{{end}}",
          "wipeFilesystem": true,
          "label": "kubelet",
          "format": "xfs"
        }
      }{{ if eq .InstanceRole "master" -}},
      {
        "name": "etcd",
        "mount": {
          "device": "/dev/disk/azure/scsi1/lun0",
          "wipeFilesystem": false,
          "label": "etcd",
          "format": "ext4"
        }
      }
	   {{- end }}
    ]
  }
}
`
