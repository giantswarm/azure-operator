package ignition

const IngressLB = `apiVersion: v1
kind: Service
metadata:
  name: ingress-loadbalancer
  namespace: kube-system
  annotations:
    external-dns.alpha.kubernetes.io/hostname: ingress.{{ .ClusterDNSDomain }}.
    # this annotation adds lb rules fot both TCP and UDP to allow UDP outbound connection with Standard LB
    service.beta.kubernetes.io/azure-load-balancer-mixed-protocols: "true" 
    # this annotation re-uses already allocated public IP for ingress LB.
    service.beta.kubernetes.io/azure-pip-name: {{ .PublicIPName }}
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
