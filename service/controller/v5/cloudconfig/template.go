package cloudconfig

const (
	calicoAzureFileName       = "/srv/calico-azure.yaml"
	calicoAzureFileOwner      = "root:root"
	calicoAzureFilePermission = 0600
	calicoAzureFileTemplate   = `# Extra changes:
#  - Added resource limits to calico-node.
#  - Added resource limits to install-cni.
#  - Made install-cni initContainer.
#  - Added 'priorityClassName: system-cluster-critical' to calico daemonset and calico typha deployment.
#
# Calico Version v3.5.1
# https://docs.projectcalico.org/v3.2/releases#v3.5.1
# This manifest includes the following component versions:
#   calico/node:v3.5.1
#   calico/cni:v3.5.1

# This ConfigMap is used to configure a self-hosted Calico installation.
kind: ConfigMap
apiVersion: v1
metadata:
  name: calico-config
  namespace: kube-system
data:
  # To enable Typha, set this to "calico-typha" *and* set a non-zero value for Typha replicas
  # below.  We recommend using Typha if you have more than 50 nodes. Above 100 nodes it is
  # essential.
  typha_service_name: "none"

  # The CNI network configuration to install on each node.  The special
  # values in this config will be automatically populated.
  cni_network_config: |-
    {
      "name": "k8s-pod-network",
      "cniVersion": "0.3.0",
      "plugins": [
        {
          "type": "calico",
          "log_level": "info",
          "datastore_type": "kubernetes",
          "nodename": "__KUBERNETES_NODE_NAME__",
          "mtu": 1500,
          "ipam": {
            "type": "host-local",
            "subnet": "usePodCidr"
          },
          "policy": {
              "type": "k8s"
          },
          "kubernetes": {
              "kubeconfig": "__KUBECONFIG_FILEPATH__"
          }
        },
        {
          "type": "portmap",
          "snat": true,
          "capabilities": {"portMappings": true}
        }
      ]
    }

---


# This manifest creates a Service, which will be backed by Calico's Typha daemon.
# Typha sits in between Felix and the API server, reducing Calico's load on the API server.

apiVersion: v1
kind: Service
metadata:
  name: calico-typha
  namespace: kube-system
  labels:
    k8s-app: calico-typha
spec:
  ports:
    - port: 5473
      protocol: TCP
      targetPort: calico-typha
      name: calico-typha
  selector:
    k8s-app: calico-typha

---

# This manifest creates a Deployment of Typha to back the above service.

apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: calico-typha
  namespace: kube-system
  labels:
    k8s-app: calico-typha
spec:
  # Number of Typha replicas.  To enable Typha, set this to a non-zero value *and* set the
  # typha_service_name variable in the calico-config ConfigMap above.
  #
  # We recommend using Typha if you have more than 50 nodes.  Above 100 nodes it is essential
  # (when using the Kubernetes datastore).  Use one replica for every 100-200 nodes.  In
  # production, we recommend running at least 3 replicas to reduce the impact of rolling upgrade.
  replicas: 0
  revisionHistoryLimit: 2
  template:
    metadata:
      labels:
        k8s-app: calico-typha
      annotations:
        # This, along with the CriticalAddonsOnly toleration below, marks the pod as a critical
        # add-on, ensuring it gets priority scheduling and that its resources are reserved
        # if it ever gets evicted.
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      hostNetwork: true
      tolerations:
        # Mark the pod as a critical add-on for rescheduling.
        - key: CriticalAddonsOnly
          operator: Exists
      # Since Calico can't network a pod until Typha is up, we need to run Typha itself
      # as a host-networked pod.
      serviceAccountName: calico-node
      priorityClassName: system-cluster-critical
      containers:
      - image: quay.io/giantswarm/typha:v3.2.3
        name: calico-typha
        ports:
        - containerPort: 5473
          name: calico-typha
          protocol: TCP
        env:
          # Enable "info" logging by default.  Can be set to "debug" to increase verbosity.
          - name: TYPHA_LOGSEVERITYSCREEN
            value: "info"
          # Disable logging to file and syslog since those don't make sense in Kubernetes.
          - name: TYPHA_LOGFILEPATH
            value: "none"
          - name: TYPHA_LOGSEVERITYSYS
            value: "none"
          # Monitor the Kubernetes API to find the number of running instances and rebalance
          # connections.
          - name: TYPHA_CONNECTIONREBALANCINGMODE
            value: "kubernetes"
          - name: TYPHA_DATASTORETYPE
            value: "kubernetes"
          - name: TYPHA_HEALTHENABLED
            value: "true"
          # Uncomment these lines to enable prometheus metrics.  Since Typha is host-networked,
          # this opens a port on the host, which may need to be secured.
          #- name: TYPHA_PROMETHEUSMETRICSENABLED
          #  value: "true"
          #- name: TYPHA_PROMETHEUSMETRICSPORT
          #  value: "9093"
        livenessProbe:
          exec:
            command:
            - calico-typha
            - check
            - liveness
          periodSeconds: 30
          initialDelaySeconds: 30
        readinessProbe:
          exec:
            command:
            - calico-typha
            - check
            - readiness
          periodSeconds: 10

---

# This manifest installs the calico/node container, as well
# as the Calico CNI plugins and network config on
# each master and worker node in a Kubernetes cluster.
kind: DaemonSet
apiVersion: extensions/v1beta1
metadata:
  name: calico-node
  namespace: kube-system
  labels:
    k8s-app: calico-node
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        k8s-app: calico-node
      annotations:
        # This, along with the CriticalAddonsOnly toleration below,
        # marks the pod as a critical add-on, ensuring it gets
        # priority scheduling and that its resources are reserved
        # if it ever gets evicted.
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
      hostNetwork: true
      tolerations:
        # Make sure calico-node gets scheduled on all nodes.
        - effect: NoSchedule
          operator: Exists
        # Mark the pod as a critical add-on for rescheduling.
        - key: CriticalAddonsOnly
          operator: Exists
        - effect: NoExecute
          operator: Exists
      serviceAccountName: calico-node
      priorityClassName: system-cluster-critical
      # Minimize downtime during a rolling upgrade or deletion; tell Kubernetes to do a "force
      # deletion": https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods.
      terminationGracePeriodSeconds: 0
      containers:
        # Runs calico/node container on each Kubernetes node.  This
        # container programs network policy and routes on each
        # host.
        - name: calico-node
          image: quay.io/giantswarm/node:v3.5.1
          env:
            # Use Kubernetes API as the backing datastore.
            - name: DATASTORE_TYPE
              value: "kubernetes"
            # Typha support: controlled by the ConfigMap.
            - name: FELIX_TYPHAK8SSERVICENAME
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: typha_service_name
            # Wait for the datastore.
            - name: WAIT_FOR_DATASTORE
              value: "true"
            # Set based on the k8s node name.
            - name: NODENAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            # Don't enable BGP.
            - name: CALICO_NETWORKING_BACKEND
              value: "none"
            # Cluster type to identify the deployment type
            - name: CLUSTER_TYPE
              value: "k8s"
            # The default IPv4 pool to create on startup if none exists. Pod IPs will be
            # chosen from this range. Changing this value after installation will have
            # no effect. This should fall within --cluster-cidr.
            - name: CALICO_IPV4POOL_CIDR
              value: "{{ .CalicoCIDR }}"
            # Disable file logging so kubectl logs works.
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            # Set Felix endpoint to host default action to ACCEPT.
            - name: FELIX_DEFAULTENDPOINTTOHOSTACTION
              value: "ACCEPT"
            # Disable IPv6 on Kubernetes.
            - name: FELIX_IPV6SUPPORT
              value: "false"
            # Set Felix logging to "info"
            - name: FELIX_LOGSEVERITYSCREEN
              value: "info"
            - name: FELIX_HEALTHENABLED
              value: "true"
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 250m
              memory: 150Mi
            limits:
              cpu: 250m
              memory: 150Mi
          livenessProbe:
            httpGet:
              path: /liveness
              port: 9099
              host: localhost
            periodSeconds: 10
            initialDelaySeconds: 10
            failureThreshold: 6
          readinessProbe:
            exec:
              command:
              - /bin/calico-node
              - -felix-ready
            periodSeconds: 10
          volumeMounts:
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
            - mountPath: /var/lib/calico
              name: var-lib-calico
              readOnly: false
      initContainers:
        # This container installs the Calico CNI binaries
        # and CNI network config file on each node.
        - name: install-cni
          image: quay.io/giantswarm/cni:v3.5.1
          command: ["/install-cni.sh"]
          env:
            # Name of the CNI config file to create.
            - name: CNI_CONF_NAME
              value: "10-calico.conflist"
            # Set the hostname based on the k8s node name.
            - name: KUBERNETES_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            # The CNI network config to install on each node.
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: cni_network_config
            # Prevents the container from sleeping forever.
            - name: SLEEP
              value: "false"
          resources:
            requests:
              cpu: 50m
              memory: 100Mi
            limits:
              cpu: 50m
              memory: 100Mi
          volumeMounts:
            - mountPath: /host/opt/cni/bin
              name: cni-bin-dir
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
      volumes:
        # Used by calico/node.
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        - name: var-lib-calico
          hostPath:
            path: /var/lib/calico
        # Used to install CNI.
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
---

apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-node
  namespace: kube-system

---

# Create all the CustomResourceDefinitions needed for
# Calico policy-only mode.

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
   name: felixconfigurations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: FelixConfiguration
    plural: felixconfigurations
    singular: felixconfiguration
---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: bgpconfigurations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: BGPConfiguration
    plural: bgpconfigurations
    singular: bgpconfiguration

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ippools.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: IPPool
    plural: ippools
    singular: ippool

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: hostendpoints.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: HostEndpoint
    plural: hostendpoints
    singular: hostendpoint

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: clusterinformations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: ClusterInformation
    plural: clusterinformations
    singular: clusterinformation

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: globalnetworkpolicies.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: GlobalNetworkPolicy
    plural: globalnetworkpolicies
    singular: globalnetworkpolicy

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: globalnetworksets.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: GlobalNetworkSet
    plural: globalnetworksets
    singular: globalnetworkset

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: networkpolicies.crd.projectcalico.org
spec:
  scope: Namespaced
  group: crd.projectcalico.org
  version: v1
  names:
    kind: NetworkPolicy
    plural: networkpolicies
    singular: networkpolicy

---

# Calico Version v3.5.1
# https://docs.projectcalico.org/v3.5/releases#v3.5.1
# Include a clusterrole for the calico-node DaemonSet,
# and bind it to the calico-node serviceaccount.
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: calico-node
rules:
  # The CNI plugin needs to get pods, nodes, and namespaces.
  - apiGroups: [""]
    resources:
      - pods
      - nodes
      - namespaces
    verbs:
      - get
  - apiGroups: [""]
    resources:
      - endpoints
      - services
    verbs:
      # Used to discover service IPs for advertisement.
      - watch
      - list
      # Used to discover Typhas.
      - get
  - apiGroups: [""]
    resources:
      - nodes/status
    verbs:
      # Needed for clearing NodeNetworkUnavailable flag.
      - patch
      # Calico stores some configuration information in node annotations.
      - update
  # Watch for changes to Kubernetes NetworkPolicies.
  - apiGroups: ["networking.k8s.io"]
    resources:
      - networkpolicies
    verbs:
      - watch
      - list
  # Used by Calico for policy information.
  - apiGroups: [""]
    resources:
      - pods
      - namespaces
      - serviceaccounts
    verbs:
      - list
      - watch
  # The CNI plugin patches pods/status.
  - apiGroups: [""]
    resources:
      - pods/status
    verbs:
      - patch
  # Calico monitors various CRDs for config.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - globalfelixconfigs
      - felixconfigurations
      - bgppeers
      - globalbgpconfigs
      - bgpconfigurations
      - ippools
      - globalnetworkpolicies
      - globalnetworksets
      - networkpolicies
      - clusterinformations
      - hostendpoints
    verbs:
      - get
      - list
      - watch
  # Calico must create and update some CRDs on startup.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - ippools
      - felixconfigurations
      - clusterinformations
    verbs:
      - create
      - update
  # Calico stores some configuration information on the node.
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - get
      - list
      - watch
  # These permissions are only requried for upgrade from v2.6, and can
  # be removed after upgrade or on fresh installations.
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - bgpconfigurations
      - bgppeers
    verbs:
      - create
      - update
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: calico-node
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: calico-node
subjects:
- kind: ServiceAccount
  name: calico-node
  namespace: kube-system
---
`
)

const (
	certFileOwner      = "root:root"
	certFilePermission = 0600
)

const (
	// Below we need separate path for master's kubelet as master machine
	// is regular VM not running in VMSS. This may change soonish.

	cloudProviderConfFileName       = "/etc/kubernetes/config/azure.yaml"
	cloudProviderConfFileOwner      = "root:root"
	cloudProviderConfFilePermission = 0600
	cloudProviderConfFileTemplate   = `cloud: {{ .Cloud }}
tenantId: {{ .TenantID }}
subscriptionId: {{ .SubscriptionID }}
resourceGroup: {{ .ResourceGroup }}
location: {{ .Location }}
{{- if not .UseManagedIdentityExtension }}
aadClientId: {{ .AADClientID }}
aadClientSecret: {{ .AADClientSecret }}
{{- end}}
primaryScaleSetName: {{ .PrimaryScaleSetName }}
subnetName: {{ .SubnetName }}
securityGroupName: {{ .SecurityGroupName }}
vnetName: {{ .VnetName }}
vmType: vmss
routeTableName: {{ .RouteTableName }}
useManagedIdentityExtension: {{ .UseManagedIdentityExtension }}
`
)

const (
	defaultStorageClassFileName       = "/srv/default-storage-class.yaml"
	defaultStorageClassFileOwner      = "root:root"
	defaultStorageClassFilePermission = 0600
	defaultStorageClassFileTemplate   = `apiVersion: storage.k8s.io/v1beta1
kind: StorageClass
metadata:
  name: default
  annotations:
    storageclass.beta.kubernetes.io/is-default-class: "true"
  labels:
    kubernetes.io/cluster-service: "true"
provisioner: kubernetes.io/azure-disk
parameters:
  kind: Managed
  storageaccounttype: Premium_LRS
---
apiVersion: storage.k8s.io/v1beta1
kind: StorageClass
metadata:
  name: managed-premium
  annotations:
  labels:
    kubernetes.io/cluster-service: "true"
provisioner: kubernetes.io/azure-disk
parameters:
  kind: Managed
  storageaccounttype: Premium_LRS
---
apiVersion: storage.k8s.io/v1beta1
kind: StorageClass
metadata:
  name: managed-standard
  annotations:
  labels:
    kubernetes.io/cluster-service: "true"
provisioner: kubernetes.io/azure-disk
parameters:
  kind: Managed
  storageaccounttype: Standard_LRS
`
)

const (
	etcdMountUnitName     = "var-lib-etcd.mount"
	etcdMountUnitTemplate = `[Unit]
Description=Mounts disk to /var/lib/etcd
Requires=format-etcd-disk.service
After=format-etcd-disk.service
Before=etcd3.service

[Mount]
What=/dev/disk/by-label/etcd
Where=/var/lib/etcd
Type=ext4

[Install]
WantedBy=multi-user.target
`
	etcdDiskFormatUnitName     = "format-etcd-disk.service"
	etcdDiskFormatUnitTemplate = `[Unit]
Description=Formats the disk drive
Requires=dev-disk-azure-scsi1-lun{{ .LUNID }}.device
After=dev-disk-azure-scsi1-lun{{ .LUNID }}.device

[Service]
Type=oneshot
RemainAfterExit=yes
Environment="LABEL=etcd"
Environment="DEV=/dev/disk/azure/scsi1/lun{{ .LUNID }}"
# Do not wipe the disk if it's already being used, so the etcd data is persistent across reboot.
ExecStart=-/bin/bash -c "if ! findfs LABEL=$LABEL > /tmp/label.$LABEL; then wipefs -a -f $DEV && mkfs.ext4 -T news -F -L $LABEL $DEV && echo wiped; fi"

[Install]
WantedBy=multi-user.target
`
	dockerMountUnitName     = "var-lib-docker.mount"
	dockerMountUnitTemplate = `[Unit]
Description=Mounts disk to /var/lib/docker
Requires=format-docker-disk.service
After=format-docker-disk.service
Before=docker.service

[Mount]
What=/dev/disk/by-label/docker
Where=/var/lib/docker
Type=xfs

[Install]
WantedBy=multi-user.target
`
	dockerDiskFormatUnitName     = "format-docker-disk.service"
	dockerDiskFormatUnitTemplate = `[Unit]
Description=Formats the disk drive
Requires=dev-disk-azure-scsi1-lun{{ .LUNID }}.device
After=dev-disk-azure-scsi1-lun{{ .LUNID }}.device

[Service]
Type=oneshot
RemainAfterExit=yes
Environment="LABEL=docker"
Environment="DEV=/dev/disk/azure/scsi1/lun{{ .LUNID }}"
ExecStart=-/bin/bash -c "wipefs -a -f $DEV && mkfs.xfs -L $LABEL $DEV"

[Install]
WantedBy=multi-user.target
`
)

// TODO: This part should be removed when parameterized charts will be supported.
const (
	ingressLBFileName       = "/srv/k8s-ingress-loadbalancer.yaml"
	ingressLBFileOwner      = "root:root"
	ingressLBFilePermission = 0600
	ingressLBFileTemplate   = `apiVersion: v1
kind: Service
metadata:
  name: ingress-loadbalancer
  namespace: kube-system
  annotations:
    external-dns.alpha.kubernetes.io/hostname: ingress.{{ .ClusterDNSDomain }}.
  labels:
    app: ingress-loadbalancer
spec:
  type: LoadBalancer
  ports:
  - name: http
    port: 80
  - name: https
    port: 443
  selector:
    k8s-app: nginx-ingress-controller
`
	ingressLBUnitName     = "k8s-addons-ingress-loadbalancer.service"
	ingressLBUnitTemplate = `[Unit]
Description=Add Kubernetes load balancer manifest to addons
ConditionPathExists=!/srv/k8s-ingress-loadbalancer.yaml.lock
[Service]
Type=oneshot
ExecStart=/bin/sh -c "sed -i '/MANIFESTS=\"\"/a MANIFESTS=\"${MANIFESTS} k8s-ingress-loadbalancer.yaml\"' /opt/k8s-addons && touch /srv/k8s-ingress-loadbalancer.yaml.lock"
[Install]
WantedBy=multi-user.target
`
)

// Certificate decryption
const (
	certDecrypterUnitName     = "certificate-decrypter.service"
	certDecrypterUnitTemplate = `[Unit]
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
)
