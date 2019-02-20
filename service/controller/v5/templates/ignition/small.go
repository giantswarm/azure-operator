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
    ]
  }
 }
`
