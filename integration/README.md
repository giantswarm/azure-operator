# How to run integration test

```
$ minikube delete
$ minikube start --extra-config=apiserver.Authorization.Mode=RBAC
$ e2e-harness setup --remote=false
$ e2e-harness test --test-dir=integration/test/mytest
```
