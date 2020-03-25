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
          "device": "/dev/disk/azure/scsi1/lun61",
          "wipeFilesystem": true,
          "label": "docker",
          "format": "xfs"
        }
      },
      { 
        "name": "kubelet",
        "mount": {
          "device": "/dev/disk/azure/scsi1/lun62",
          "wipeFilesystem": true,
          "label": "kubelet",
          "format": "xfs"
        }
      }{{ if eq .InstanceRole "master" -}},
      {
        "name": "etcd",
        "mount": {
          "device": "/dev/disk/azure/scsi1/lun63",
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
