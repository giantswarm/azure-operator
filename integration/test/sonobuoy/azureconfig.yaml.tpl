apiVersion: provider.giantswarm.io/v1alpha1
kind: AzureConfig
metadata:
  name: ci-wip-sonobuoy-${CLUSTER_ID}
  namespace: default
spec:
  azure:
    availabilityZones:
    - 1
    - 2
    - 3
    credentialSecret:
      name: credential-default
      namespace: giantswarm
    dnsZones:
      api:
        name: gollum.westeurope.azure.gigantic.io
        resourceGroup: gollum
      etcd:
        name: gollum.westeurope.azure.gigantic.io
        resourceGroup: gollum
      ingress:
        name: gollum.westeurope.azure.gigantic.io
        resourceGroup: gollum
    masters:
    - dockerVolumeSizeGB: 50
      kubeletVolumeSizeGB: 100
      vmSize: Standard_D4s_v3
    virtualNetwork:
      calicoSubnetCIDR: 10.1.128.0/17
      cidr: 10.1.0.0/16
      masterSubnetCIDR: 10.1.0.0/24
      workerSubnetCIDR: 10.1.1.0/24
    workers:
    - dockerVolumeSizeGB: 50
      kubeletVolumeSizeGB: 100
      vmSize: Standard_D4s_v3
    - dockerVolumeSizeGB: 50
      kubeletVolumeSizeGB: 100
      vmSize: Standard_D4s_v3
    - dockerVolumeSizeGB: 50
      kubeletVolumeSizeGB: 100
      vmSize: Standard_D4s_v3
  cluster:
    calico:
      cidr: 0
      mtu: 1430
      subnet: ""
    customer:
      id: demo
    docker:
      daemon:
        cidr: 172.17.0.1/16
    etcd:
      altNames: ""
      domain: etcd.${CLUSTER_ID}.k8s.gollum.westeurope.azure.gigantic.io
      port: 2379
      prefix: giantswarm.io
    id: ${CLUSTER_ID}
    kubernetes:
      api:
        clusterIPRange: 172.31.0.0/16
        domain: api.${CLUSTER_ID}.k8s.gollum.westeurope.azure.gigantic.io
        securePort: 443
      cloudProvider: azure
      dns:
        ip: 172.31.0.10
      domain: cluster.local
      ingressController:
        docker:
          image: quay.io/giantswarm/nginx-ingress-controller:0.9.0-beta.11
        domain: ingress.${CLUSTER_ID}.k8s.gollum.westeurope.azure.gigantic.io
        insecurePort: 30010
        securePort: 30011
        wildcardDomain: '*.${CLUSTER_ID}.k8s.gollum.westeurope.azure.gigantic.io'
      kubelet:
        altNames: kubernetes,kubernetes.default,kubernetes.default.svc,kubernetes.default.svc.cluster.local
        domain: worker.${CLUSTER_ID}.k8s.gollum.westeurope.azure.gigantic.io
        labels: azure-operator.giantswarm.io/version=2.9.0,giantswarm.io/provider=azure
        port: 10250
      networkSetup:
        docker:
          image: quay.io/giantswarm/k8s-setup-network-environment:1f4ffc52095ac368847ce3428ea99b257003d9b9
      ssh:
        userList:
        - name: giantswarm
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCuJvxy3FKGrfJ4XB5exEdKXiqqteXEPFzPtex6dC0lHyigtO7l+NXXbs9Lga2+Ifs0Tza92MRhg/FJ+6za3oULFo7+gDyt86DIkZkMFdnSv9+YxYe+g4zqakSV+bLVf2KP6krUGJb7t4Nb+gGH62AiUx+58Onxn5rvYC0/AXOYhkAiH8PydXTDJDPhSA/qWSWEeCQistpZEDFnaVi0e7uq/k3hWJ+v9Gz0qqChHKWWOYp3W6aiIE3G6gLOXNEBdWRrjK6xmrSmo9Toqh1G7iIV0Y6o9w5gIHJxf6+8X70DCuVDx9OLHmjjMyGnd+1c3yTFMUdugtvmeiGWE0E7ZjNSNIqWlnvYJ0E1XPBiyQ7nhitOtVvPC4kpRP7nOFiCK9n8Lr3z3p4v3GO0FU3/qvLX+ECOrYK316gtwSJMd+HIouCbaJaFGvT34peaq1uluOP/JE+rFOnszZFpCYgTY2b4lWjf2krkI/a/3NDJPnRpjoE3RjmbepkZeIdOKTCTH1xYZ3O8dWKRX8X4xORvKJO+oV2UdoZlFa/WJTmq23z4pCVm0UWDYR5C2b9fHwxh/xrPT7CQ0E+E9wmeOvR4wppDMseGQCL+rSzy2AYiQ3D8iQxk0r6T+9MyiRCfuY73p63gB3m37jMQSLHvm77MkRnYcBy61Qxk+y+ls2D0xJfqxw==
            giantswarm
        - name: joe
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDE178xsxTfHERTXpzxbd8AsH4l1kQ+2y2+s1Ed0YQTfbNzCHMBKuCmabyv56QISc0Frp6oFNutmbRQpRlxNRzvWvcdapb2+wNQIOpc6/aQBPbiyCdU6Tjcw1p3p7z/O8M9wIPZ2e9zYyUjV0EzN/iZxrdDBduF1mrAjKzeG9E+McEUaD3LIJCxmljrt3248wusHvdwpLJGTM8K8ajdrlKNET9KEI3lWTaHBxr8v/cPixBJb+rxnMZuBRV/Hc3XN13OhW3wVftGMkgjrS0oVTcXE8YlrCYCNNlw+A1hVHZ3XBbV/g1Ww65lmL2AOHrOlnUd96bbocFcm6btqUuWr1clDfEZ/FvfAvWKe9pZb2rCxqOCnLzZmB6zUPj9dS8Cg7nnXZFfsIE0p71sO2i4cYd0l9uzQpmsxYPAy+BAdRpR9P2oM1CM/DbLjlO5IIb9qTB3O4R2zaG5WpVjAdvqo9XptXKa5uIi8ZoVHvhCdnqskwaXsMpEHavQVvdxPBal01smXxFv6lLqKMVkzJRBkXBEWXvxa12pv2kiFnaxMWK95jqLFHXpjZVrYS1Z77ld9+SXGr0KjAvd6SShPg1ggiDAd4suBDUbeyVQyhzr0CGJ4auiqHsO5IDSdaFFo7xeqnBzAT+jxsBfzKhn9In7HZNf1XnG+2fF41eqnobWwMbCaQ==  joe
        - name: marian
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDMSBIK7+MCULpHyUhLvYOKgQEyhkBR1n7/5oyKhzAD5pvAjqBzBcQ64S4EYxL0/Y/JDa2bMzAEzaw0VL1e2kz/nAfd7QiW3ZgyU6uYGDwHHWDekAY+Q30giQoqP3QxFSDTjUVb1EC4kIO49uzAwItwM2ah6C/Jmz4/EWMP+2MKrwCe8DUTCYPI0RyXpyj0O0Uz+11VGVCIdMbxq3O62giC4WwNUFC+RDGS4plrsOo4whrLOlE7ZrYjSp1dU+GdNQmrKXJA8j9k8asIsChljrx6wF2aS7gMF5ltj8M3ufk1Cz4FN8/5luAE0qx14I8K0yej8Ann4dohrRm8sPz3aQOh
            marian
        - name: puja
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDFKCoKe/Z2cT7duLiiPdgUXKcsHx+3ESCa0t4hOZtfw6BHptJ0dpDTAqbkvpGRwErVpOO9tIQittWnzgX0RLqnDB98I7eZQ04AIZwoW9AX2aMEyLgbVMTeG9Xgy79pefjVV75T2lVjXtcOY2wJtf6ZU1KFq8/dHZb7vYzzobHBq9j4vIB5ZsNI94jJm7I6TLr24vga3+MEgQrsEQdRqZ2vxacSU1h+LSdfseGQew1XlxSTfTfglUcXE0WUlEFnak9z0JwQbblEmKQsinIwO4O0Sk6FQXObCFlss//gubk64/OM/87I/aKjrmbQTRMkxyqJ5jO4yIXOxeHpp5kNA9AKSmgHABhr1ViS6ocWO8mMekbLdxDWMdViTR6XxtFSPUCgTFAirsQi6/9qfV+6u2RLhKihuajy8akFi4BYqSGr17/crrkCYydBJRUIqNmQSdzGKodTJ9d556iFZ3rCM+Xe2mm4KsHkIQ3YphPMzb0yAWEtZl1ncdqSXHz74M9b1KHUzyJgQhv5KOzhURxXR/UVBy7NNPae8XSEFId/O2uHgc/mWV5Xr5ZwxbwXsmlyto53+EmgynnPcL96RgVyiAmHL/vtvOGAzVSOPtNsnU7QG/YDfA2WrxLmuGEA0WzC63iXZCqSFbPK0adelJo9vctCB2gozrVpjqXWpskg8SZMqw==
            puja
        - name: teemow
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCvDv/5/bOgnuNXbxB7n8n8reDpRdAPeZvuCE3pJPVYJNfL0hoNzGhw4MBLRivSd9ViMM9X1CE7iVK6RFO46yt4GK5JWYi1UM88q6I+4bcnswUAWsIFX5a/U0LK0wIR3akDDyU9WEA7CXoJ7IxF7dtIYC9OIqrD6gXc4P/UI0jZQI5iZY3qNjlKVAwsNz8pD/BE2sPpNVHumzgcLJEveoc3WMCmfBAAWQAMfRlhlJ5LjM7Py/5k1/s5Myn4L/yoAvxMWev4k2ZYpumt752/r927K7AIrK/OYTfqLPKzZYSLWAj4g7L3/65sKpFm6g+HFgDmlScgf9P4bAoLn6+mWbYL
            teemow
        - name: tobias
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCS/thnoYaUYmuDxwQF9ES3Jsq6KltO+QU/8PVo1tUr5vlEfEY1Q9JYHiPKJW+U0cMH3a/Jv2IDTaH629djoNdtTottaYGDINBoVIlAdR+vwm3JzVUB02mb+QXTzhzLc58fdwhHN0PS82/BcSSFpQzI7PedRGMtzS6Qxcx4YfrzC16vsdF8wMw+tVbtI2fDLwfd9NcpCDr582NrX/qOR22zeck3VVgppnuC5mGAC+XvHCRbp+4pZJ0W4fpEIGwG1cPbktvdA0wYcn7GJo7fU11066PMGMXplV+DEnQTpBUbP+NFXRuY7RzTeuTGSZHXsWO11cmpLPVVB7LdAaQuSPi1
            tobias
        - name: xh3b4sd
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQClCCgsKl7+mQwD+giN6OEruV1ur/prpWXfyGHJyGGQkROZA3IcrpmRPWmKKXpCaW+G8lcb9DXD/K7/rNAh+4hpsfvCUs8u0mJ6u4El/8dcRTQaZUdLX8q3AZZ38gmk+yZz241x7LGd05D4H+aq9sVdtbcAepINUJyZ7p3yXTfCYwHC7QMYiuRFKMaUHY50shFhSYdD9TCEFtH2ybPi1/WOCX6gf90f6O0Ivo7tzwtYGV8ToIa2nO+CqwlIRiGqEy4/g9h1gCPDvgcLZmok74V6mH12whNdMDyJyuT8S1dLwNiKoYkvMbcUkpE0O/0LBCg+SsHVHmgnsNx9t0hUg8iR
            xh3b4sd
        - name: calvix
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC9IyAZvlEL7lrxDghpqWjs/z/q4E0OtEbmKW9oD0zhYfyHIaX33YYoj3iC7oEd6OEvY4+L4awjRZ2FrXerN/tTg9t1zrW7f7Tah/SnS9XYY9zyo4uzuq1Pa6spOkjpcjtXbQwdQSATD0eeLraBWWVBDIg1COAMsAhveP04UaXAKGSQst6df007dIS5pmcATASNNBc9zzBmJgFwPDLwVviYqoqcYTASka4fSQhQ+fSj9zO1pgrCvvsmA/QeHz2Cn5uFzjh8ftqkM10sjiYibknsBuvVKZ2KpeTY6XoTOT0d9YWoJpfqAEE00+RmYLqDTQGWm5pRuZSc9vbnnH2MiEKf
            calvix
        - name: rossf7
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCY0Rn2VqhtOFy7LY6nu53c9bP5Fy0KQ+P5/MA/s4aG8+veIqTyhhpHLwPF0hKbi1mq+HaFvm1nLbovZcTTG2Z+BoP/Y5kV5EOjnq415EtT3rH0YdM0h69Qxuc0KqUvU/F43XOhpEH0o8L+ZK+Vq4UrRPIDRjftc8N5h6MJszAow/kiC7d30nYsPio6FuWHH5jZaAKjucQbBsDU5r160mtVk0HCexutm3s3fHTADojZjFA3t8FJy7vO+Og+VDVzV9ai9E32mgytNL0wVE1dUGqPwoM9MrzxNC2TZedS74zqBoK9TL3y1sfVzD5RpdX4Z5FInhtTz1z4nnYzsPiYQKMx
            rossf7
        - name: oponder
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAxEDA/Nr7++9SoFOhTeNJMAubkzJmGZWHtXul0kM+FJR0TM/1V3b4XfoRAwU9gaG88P+4venSwcbLVfvvacnlGQ8hTgw2Jlpz6Z9+iIVjru2+nYgJeELFff5bdPLYWE4Ft/VYpwGibD1DbVGldsO3I7EdaEfd5FeOF0Fk1xPK50UGvTq9CkU+wEcTc9eDzFWpLoz/69KWG/F7XEZhWqswUjHaN1UJtuBlVmoe/0OrlyIYBl2CeUzpmJHNDDv7u8gKOCOFwmzMdDieGHzIITCovIoGLhIJS1dGdT3FPvTPG/s8VOZ/QGS9rfsc5x08J0v0JqwN8GZT9FxFJECeXTN6bw==
            oponder
        - name: kopiczko
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDzY+7kp4XTNfinVLbDo28E0yaJUMvEabzdsheGHG3+gubakJgITZw+3m4vy45WRoF8QDOjpZ12n9ov9bpz5X8kRmHSthvWYtNRYJWCJc0d3+td1/Ki9CaHesNhKdeVYcw9g5x55h6o4EPx+g6wIsBhxqjcdZ5O37M+KWXlBfLoP4WKBjORhD4kpU+suA+rMIRF/njLs8zswHL8or3Ynp+voZM1PVCfxENp9ktNeA7W2KyUZpgDtoWxN2cnj0BOs/t2w+XZhqgsPo/9zXO6C0XIvPv69MAOHYMsomKldQgpy+MlODvu/sbP5ruB/4vGiqCg0+E2KS1ZK85KytQtfai3IcmglNtHdRKyYmf2WkY/GMGB6sgQKZWJE4XeN7+mZvRrOih2yo+/GJCkI5U0eWIUInyh/lBMIxVbckbpdZYxMd2SaERkTFNDhqSgncrPu5gW7ZFejgie9gVlkYBLI/D1XXM1YzvpSlA80D9kxLhiqdaAXP7CysPgS7EN56zM0SzHR0vxrr4dhB9XuBxlMeTtC0dvaMPkiIJj43MLDNzE4wtXEpzFQnmzRoOrPAACRhr0OcOTbsH7X+QR9JUB1ygTRSN/kHVFyL+EPRRsfuzbxw57jkzIE7ihGtcINtyv4dIFfqE7c+uDDZV/yUi2C5PI6FLHW33DXwqbzR8yZuOu2Q==
            pawel
        - name: marcelmue
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC9pOt/FFXonkNDGamoGMVg6wQJYS7m9r/OO2wWoEPNQS/M4nGL/szKlZfD4Z6tKS3WyeLY5XU7QjOFb3Gt3QQAaVOSDgTkfH1i+usWSNzFlhjARQkIUs0j9m30o7sXznZNOy4r73bnfUjTwYJifGUWBPq/jGokLNCxBRPCaJFi8Od5De3DyuDd93SAoXkTJaDPr7J90tzkVLI6ek5va1GSTeVdHbAifds8r8Shm0wgdmcVKiOYt7/oyzavl0x5XPzMAVXeUI0jIopsvqjiy/fS+Cq7i1TMBQ+rkycWLc8X8CM8U84OQ6eb8LgQw0A4xqVtZHOl2FlHHtWNjLhnwO5MHWjdWxCUSEshK+tS5Wm64ph37nfObupPpoMcRRTmKB2SdsigU0T/aJ9zoJorsIBKDY6lqmXoYk07XEyu5tmuG9cuiP900yLRIoCZeQ14WP3+3KDWLfjic3W8wXr0xlLOTaPNtNpX+v6X4n8R4HuJ0zQw1znEXhBlhTEZTRe1qmdPQTNwQa6XwOlEJVYOIHEOobTQVu+ReIrj+XT3b1VR531pG8M3o9Rhq29aFpAsYiDh12aGk45b+61hddF3WybpMOVQqfvYnf1lVwt/0PujJuIC7e6WuHAlRS2Sshb19ROG8w2mW1sGbp2zN7Y+MUAI5LVOHrBtRqEoZkhz+JXMzQ==
            marcelmue
        - name: vol
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDFGDg/p4JWewXAs8kExJnCaNXEN1v2LZf0YWWiblHFp1+i2bp8qSmAJT3i6Yw0kHY2/6MotBCKAsFtlqxuhKaFs3jDcmdOugmWz4Qj7oerQ/ypJE/wZ9PY79gbK75aEKyOdVf7dUT6Ah+oSfETgpY/3a9pVZ/dSF3WBFIBw5k4YarFzcELQE4Bo4dcsLHsNrkI9Bk6gkGbTY+1TtfJmOu0bEXxXHdEq+JfW0MFssjh3I5n0DT09qDnztAvRAjjqjlyNKNt8reErV0LlvsDM5c+426Bz9JgM5vP3sD5ai8lpuH0iCBHoo9678XTKKTYbbz0s7kgXUb0vGS+GbOcaKBKmZ8a0xDpsft9+/LbmnuUic8b4c4/cRw5wSV1IYqyDqARp/d9PaJlYa22ISGnDbYmXUTsef0PhUenK9gtYrGsVhQmkqeLYiIYqwsl7+uouFMpQDmdZjY/B4fKcRA3oRGCFuwzT1vrtJL41dw9WyzM+3xnHTMFZdko9TlgDiEeu6gdpsTGJf4VALUWgXeyW/egte2im86kjMxzQuCw/aOmiYMqwZH2YfI0dS9jLuZbxePKTUounct66SrNXBrbu2d0BiPj6bl1dG6oZhwtArRnbiG5+cTakDvLhFgahTQFAT1De7o3Nr+BfjNQkVlQNKaIPUOdypiDNJE/6q/GOHVRQw==
            vol
        - name: jgsqware
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDg0b+m551vLRqsnnDrB32PrQnO41Fv62dbYYCGqjcd/if8iOuyXvxNRnbOxnsFFPVSiqY2LB8pXofEAm5MG2qupGyBvivtnq1v+6Ld9nMYTaKdz5WKhI9ypQ/jV4G1DNYrayGno13eRmGemCEnIdZeRrVxp5EfkVX0ZyJ88998Urjv6OtSLV+GSNSiIbNYyvGjLoR0dt5LCVbwbbQd1H5wXYsSoeIkKiqfS7AtMn9wDCIyM1W15yC/4UaCMEGkVfjLZB+4Y8BBfLH1vI1h43zl4EUkaq1PASDvpX0AWlclfemkK2bGEOS/UzVJsZtM73jEoSZvq/aCLe3v/0zI/5Xl
            julian
        - name: fernando
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDJu8pVpOt0FvxceHgwJpu6SVOvC+GJTDzR+zkkD/FxETVnD7XcA1KygM2PRAiIIy1tl+2PloM/67L7jmZnBve0D6Ibp98xWA4lSKcQWLcK1He30fHl4weQJ1hCL3kwD1IvbCrm85yz51w5Key9gSXLRXFKERmcgEq//KWZLGCwr3wcnsF+CJtCb+cwn7jTGS5p9UN0kH93w17uYLOiEkF5xencjsgYFonjo4ALrgtt7e9znaS2uUmh2cB425OBhPotZ45VgE3vx548nOVReIx/IHvBbdf86V73xqjNCy1FBVMSdX4KkjsC1TLZVGUs1zpGwVWT5t1J3I6hX7HI/QvkDzN1QxZ5MZApNrBRsg7Cq1wvU/hkniOJqUbjp2znlZXGtVsLBDSp1GPYJFb7/h9jCPpAtSWenp5NXu3twBqowlC//b4EIVajYLQpgmOGcHxY5PxcSbI1hNikCy+/iTES7DbwfqFXez47dPZcRXDgmNN9EGpLkDBjgLjaInkJ3IHKgRAK17+YdHScxwc2DX6HcBvHdbUzpEwRbs3ILh/uMnHp765Pqty/Unjv74aBtUOP1iCmO9/k++9aZIr4cXzoFyqFwsuyPS62fGPxZ5ZSDdsa/jMq3gGqZVj/rRko5sTdA7z7feCrwiBkS+o6S14+c03kiig/CI4qgX8RKGUutQ==
            fernando@giantswarm.io
        - name: tuommaki
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCu2V1p0cZ2N4ucug9LQMB+YMg9AQ5+aZQTdDTZ7oBuEcBuGtdnSSbcxj1lHoMYvhz6ugFVolkusRnZSakZY/XPVlwIHC56TWWrJ0hJ4sQEzCqVSHx0ZBHaMZepxCz7KSh/4KjtZFyaBC9SFwUo7kGgBYoFdClhxZsmfMsk0RneY8FjWme/cwXSaEGdaaTyOA52UOCg6Ax3nnE/gAJBsL8HgI17bFjj8og6TdPoP+33wujGHFORy8HF/m6p1I2Nm9Mp+gkG6PzdkWbF7UFci5uYHXy5IEu6uGzEPQiB5BjgfVIvZyH3VfKxmG1T2yyp4/qDQOmkjlIahpPyI00Y3SWAab7MdQXJ2hTgWFo/NP+AEdd45+PrSvTMy2k5bVl9GMntP+z+9oAhwH8OStSCJ0GBGlVG89fd0vFV1XVmLPwS8XhuhAoU1KRt6/Hc8cs7uSUiKOTY8Xn6VNUozxK137QpHBb81jU7OCcmopF9dlqoV6m18iZK1NjP4+FFxUyi5O4HI6aFrZXf7Cw5G9C8EXML3qLIMxd2pIJsu8QTw/5kC7sBtmFY/5RqW0TZ5hWuyGSuFcRan5E08Qct5rGAQ6QjJ9rZqQUPeJFcN6gEvGUam0XdeziZD6lPFUDkte9y653lIrPqBoSbsJuk/FJU/+RTSYEl+VCmaac3ru6jYV6M8w==
            tuomas
        - name: theo
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDo5PU8w8RDWYDd8SSIKOiYJCeN398PAEJApFGpPWprewBmiDGGMHHDIDV//QR+o5MhR2hBmJ+Pw1K/Edr4u0cfGlIIb6lSVdha+jDEp0l1PtqyQzubzTH3y/RDzxCakAa419N1G3DwzJHkWBxbpyqx7i/DOYcxgQP4vCGvYgkvuOkQYCNk2hfuXAl/Aucv3JXlsuNktyXQ6XKf+Twa2Bg8jIAaYUNGqgKgzMcsCElE55bxVuYeXl441CzD2fdHmXyGwo6nefN7PZ790SxQzkGM8wBpESgc7U4IPUY/dnsn4yQBYw2meontHWGLmZjrvEYxoS7Uv4o8BX8cScgVZUhRojHvNWBBcwOx+hhuPbqoqdc8IFXQLHTTa/szvY9gwlipBejj6nJrRpl3Kxw+EX4QP/loDxkykWQUoByU479Z6/gOtgAkPOe8xZblny6r3uCZyUlaYR9ht2aOEbH8bLuYBaDTPvunMIH/RSgbNxys/Dss3ZC7MJgXtoaSpb/AGdqv1Uj4AdNJJm8544AFhmR/Tky0rms3NpaSEiwO+E5ZIJiBevqPbGWkodbfKM6uydS+wrqHOR+zTNLuwhTHVnNRZ//ePBMptnzR1qbuMmaEgmqMM1HjFflUUVdDLFK6TxcdU3YPJnnWlwGk2bCjlamGjHx0hvoJVSY+Oap6o23cJQ==
            theo@giantswarm.io
        - name: ferran
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQClSTv/MNyATqUGDzmimPo2Ri5FnDLqcdv3m7Zi0S3wSZRsFV1/2dA89akJGei+Q9VLUyZedzvcgZngTMxHGquor7q49MCQmzMuAT0eIl+1l68kJD71hf9gkZiZMJnToCk9g9Qd559QWQC3qfsHdxR3MsfIcwu/Bl14j6M4Bw1C0bPTr6XTUToiukvHkS8tbbIgnUnHg/Zdp4eCxX5lRP3FXeKv7U/aAmnh1cAT008IL+wBkxvVClPOWlwx54aTdfu+HPpPVsmBm6AZhLdeb7Yrpcd5ZE53SAJuznUJ5NfO+gFj3lma+T4g0lc6L1imzXXMCKjXyo03+9Px8R8nBvxN
        - name: salvo
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDARjlE6tGtjVE52yF3zpPlNzwjM3gcIH7TF5ZIXDYhkeHUWlH3VpPNSr0rK53dHzyVzwV6s5B9I3nysKOMkoTwl2FbIrHGXwYJ3sTs0nGg9geLPQKOjwqBMrEIJ9q2imfyMfBM8IlWxKsIsoUcPXoVYzZ0LQurvAZ93Moil+U2KZZpAqPFjmytp/xJYQIvCoFXpXmbZ/9PMxc78vUXwUuUCVERKfmthLH2fS0cRdogHAGzIvmYLDRCq9if/qmQ9zhoPNFxMQJW7vBVQCZpap/Yk1HJbG2kse7w1Esl7n88JzEjSbcRWIX3KES4OtEis+uJ04cG8x3+9beTAfLSSJ0x2E7/C/weqiSYcCLzG1s4mQW+gVV4/v+fu3iXVRlIKRRBcIlSlnpBrQ+eNjscbhg2Ku4cvStQAChmkm+jlax/9uoGXsFK3PwmOknp7Fdnos/BJfV+jhqj5zqnRmPzICwTatRKCzHdYI4xaPd+qBjHSsw+zDJ2VgYA+7nKsQuouSvXgrxjHnMLzdma2RYNIXfwn0Of0k2YxqqFpEsKSE4rnXDg8mniYag7LyyHtDxVCTccCiClMrUDtqw5tAw/imjntScF8jcfM/3ZD8U23McktHyTUphOv/GMUR3wcx4i8KoQOtmG/D0/CNXU9MIP6YeONvXxmR8rCmWhWDkDDUt9EQ==
            salvo@giantswarm.io
        - name: shw
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCt8RV4mPyAl0xHbPKmDMbyu8TCv9eWhoQd7Ixd7vU2bpjKGTj3crxpAwClL7B59MxXYg0MUX5vIJHvDUWUlzvIXQhrWBf/G5ijnvHau2jCw04xVBji7LHkbiFqAs04tStYG52os5Ote5vN3yFqaFbgtng7KAbjDM2ZfJV2KW2ILmZ1zj0FIFpkTYGJ2YQ1J+LwiCB5bdFJBE4D9BGnmTy7tMub3PBZJ7WWYT5PTFj+jXkcO52Jhdd82o0sF5RIT/1wJRG70x9swWm37zQvr+3CMj6QWjxrDn6tMgl/49eRrJ5n2+3Qa6uhedfvD7csFVrwIWT4L95aRILw2fydQMOCuc3uWuE5Tj6nGQ9THvRWZ1cUCYu1Aef7jd2CzoAdQ+Z+ZLJrx70lQkq7QnDHCON/ut3fc6atxMyKhFQe4CwplgFHBPcAqOwlKrpcH6Nx34OodJDkOYtYo7ow8XAi/mkweZzi4DB/dG0FzMNf86a/qTyDPm/myHfd9BH4wWxxZS+tnyXGTgRc4qwmPbbbeLfg4KrfAw2EYZzOxqm53qjqmBzsEZQJ8Owf7m3PXAI7fOcX3NsueS9TXBXxs6Iq5sWj+XF6uq3k6V824V2eLLkFHrYcjKuGHjwakn9jQ5Y5LQblgv7FV+bQPgTU86IaUUyVGdGIWTYbdyhC0OHkeG1/+Q==
            wealdy
        - name: thomas
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC+0XcR/g2ebbzSm3iO+DgceVRm9ZpnKe6WDc+OjTSzupa4Y/nhROEQZtNEllTkkgd7lraACdlu6PPf3vuqoaA/XZ2NNYtSOJ19JvZclLKaT7QwGlD4fMgHxESeo43SSH8Hpn+aZYgLYPjBA7lSshEpv0DSpeKhoL7sD9+SonGklRjLrhpr9RlGBlds2UiuNEVrp0TpBEqWacxPrlZ9UYfuU6lGuMWC/735ShvGMusghB9q/HLd8LdwkhY/yA31Hspc03rQbCUNC27vaUEEpGV27Z02b0e9xloo5KCHNDLBNP7zCDPCwAmwzvOkrxGAWR2fabBSvCjMBUsaIzI3PkmwUm8+LJnRnbbWTtOfcsJA+L7pptZ51d8lYenKfaszgXSKE873LxEXOMfZv4Jyzn6mBayCMFYpM88gle3nl6vjwVaE4xm5mAphZsxr7+PXhHpHtFx3dvsdLRN6vnC/woV3eY+2yj1zwukvp3P+h+su1moR2DfxBPFEhZTp2jtUtaSVwm787rtDxMGlxeZ8spLv1x2tquTDfozlM88nC7Q5nqLQcp+Cow9XqFPn8shgtboWeIc9U4zJe1D0MiJSdsvn2fxn6Q7UB65yY2xeKzkb1O3c/4V76X5q/CYJkX7/E1WdHWNlm9CQd7fq6EGBh2v4TfDdkZNa0iRJ1EPs33/wDQ==
            thomas@fussell.io
        - name: cornelius
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDRMbZrl9h23nW4gt+xQKO6QoHmjQZT4H+j+m1ly7LJHQOPMkgt6LC2+5ENqaAJSscGg9/SJ97Rua0yqfYbAEvT0iAzkcO3B84QkmZo/OU5/2yN8q1iLmwX/doN/fesAma4OMJSSY2cTsV6YwBvVwgWdzpH84jtEeD2AOwK67K82yiiWarXA0tsHy6jrlxNeJiechBzYzHEOzMECCIHHDa6kGATK8+HUfunHGZgAYlot+USVSE1d+mTCiQH4/2janBdZ/XH8h2kM17DjpRThktu9xTuYepTuqoxMXGqo+AruuF5HvW7/rvKDqbOKIyvDYcR2OKqOLJhRONAMDXkPl3/Flr4egtZlCdJ/Z/iXrqsb/C4PblqZ2fc5dwvVmK9SxIAJAhFuMAT+GqwjzrJ5aORj40NvxY4a73IXwJTlqRkDvgbgByM+f/jwE4WVC9MR/z3Q0Q9bVUYvMa5CGZMS48DlOzYjIPloyDUSgEdNpQ7pXw0dDZqG8VBd8vK26/9aqk=
            jck@cornelius-pc
        - name: zach
          publicKey: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN3XklmBrM7tkunlTJAyYdF/eWy1rZj0WIh+XvXy2x0B
            zach@giantswarm.io
        - name: jakub
          publicKey: ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDOxQcAshziCtSQ1ZMazzQ9Y1KbArmHuyEM8S+nk0dx2imM7yd42eL7iFp4YlgDdRTD9IWhBcGc5nj+mv810luDQjucdq4D2qU3gI0XbYPuc5WGgQsO/28sk6IW6OpzNQW/eowKhSxa32q6QWY91Gh3QkR6L+D1Y5mUt/R55EdLx+T6BRMIbql46r8f5IvxV0a9S8ZpTneq/7rAuZzzkOu/xb4Go/OyBWHrYh8ge2wfm7Yg7SFpRmBa67KY6ZeZYX3QoF9eqFczIBsTL5MG9ffc35L4wEL6JVggOM9NTFulB9hk3HtyFFG7PHSoiznCDvK8CY/9Ee+Cn6nWrt4nMNacaCSKdE/5IJ3V6w8a3mysIcMkoeWzT9ThC0swdOl7kPhm9EdprUdE6LQAg9MINVirG6d8NlavfH3fHAQJczo21GrOEuVLP/7fB8SluXOk7MHF3GaYC9RP//xkOWmKwHZ+vTs3yhBlQcw5baRSAVH72HLoUQggIMZKuEFRyqTZW9nPgojM416br5OE1VmfhQl3g0d7xGzXstmordlZwaeyIM7o1b4BOYQ3g8zCg62gsV8BMrIeVt5aZOKMc99Fxy2PdBhkRig+hQn8NxfkpwZElQ4XKULChKFgDstfkdO6uXFcAoLPNqOL6wXvWacRffoO3P7X5FWer51l++L01fo1Uw==
            jakub@giantswarm.io
    masters:
    - id: 3z569
    scaling:
      max: 3
      min: 3
    version: ""
    workers:
    - id: zw42i
    - id: 9b330
    - id: yh6ax
  versionBundle:
    version: 2.9.0
