package ignition

const IngressLB = `apiVersion: v1
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
