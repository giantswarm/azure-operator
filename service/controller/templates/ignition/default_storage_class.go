package ignition

const DefaultStorageClass = `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: default
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
  labels:
    kubernetes.io/cluster-service: "true"
provisioner: disk.csi.azure.com
parameters:
  cachingMode: ReadOnly
  kind: managed
  skuname: Premium_LRS
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: managed-premium
  annotations:
  labels:
    kubernetes.io/cluster-service: "true"
provisioner: disk.csi.azure.com
parameters:
  cachingMode: ReadOnly
  kind: managed
  skuname: Premium_LRS
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: managed-standard
  annotations:
  labels:
    kubernetes.io/cluster-service: "true"
provisioner: disk.csi.azure.com
parameters:
  cachingMode: ReadOnly
  kind: managed
  skuname: Standard_LRS
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:azure-files-cloud-provider
rules:
- apiGroups: ['']
  resources: ['secrets']
  verbs:     ['get','create']
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:azure-files-cloud-provider
roleRef:
  kind: ClusterRole
  apiGroup: rbac.authorization.k8s.io
  name: system:azure-files-cloud-provider
subjects:
- kind: ServiceAccount
  name: persistent-volume-binder
  namespace: kube-system
---
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: af-premium
provisioner: file.csi.azure.com
mountOptions:
  - dir_mode=0755
  - file_mode=0755
  - uid=1000
  - gid=1000
parameters:
  skuName: Premium_LRS
  fsType: ext4
allowVolumeExpansion: true
reclaimPolicy: Delete
volumeBindingMode: Immediate
---
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: af-standard
provisioner: file.csi.azure.com
mountOptions:
  - dir_mode=0755
  - file_mode=0755
  - uid=1000
  - gid=1000
parameters:
  skuName: Standard_LRS
  fsType: ext4
allowVolumeExpansion: true
reclaimPolicy: Delete
volumeBindingMode: Immediate
`
