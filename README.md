[![CircleCI](https://circleci.com/gh/giantswarm/azure-operator.svg?style=svg)](https://circleci.com/gh/giantswarm/azure-operator) [![Docker Repository on Quay](https://quay.io/repository/giantswarm/azure-operator/status "Docker Repository on Quay")](https://quay.io/repository/giantswarm/azure-operator)

# azure-operator

The azure-operator manages Kubernetes clusters running in Giantnetes on Azure.

The azure-operator is still under development. See our [aws-operator](https://github.com/giantswarm/aws-operator)
for launching Giantnetes clusters on AWS.

## Getting Project

Clone the git repository: https://github.com/giantswarm/azure-operator.git

### How to build

Build it using the standard `go build` command.

```
go build github.com/giantswarm/azure-operator
```

## Running azure-operator

Create an azure-operator role (replace SUBSCRIPTION_ID):

```bash
az role definition create --role-definition '{"Name":"azure-operator","Description":"Role for github.com/giantswarm/azure-operator","Actions":["*"],"NotActions":["Microsoft.Authorization/elevateAccess/Action"],"AssignableScopes":["/subscriptions/${SUBSCRIPTION_ID}"]}'
```

If you have a service proivder you want to reuse add the azure-operator role
(replace CLIENT_ID):

```bash
az role assignment create --assignee ${CLIENT_ID} --role azure-operator
```

Otherwise create a service provider with the azure-operator role (replace
SUBSCRIPTION_ID):

```bash
az ad sp create-for-rbac -n azure-operator-sp --role="azure-operator" --scopes="/subscriptions/${SUBSCRIPTION_ID}"
```

Follow [this guide][examples-local].

[examples-local]: https://github.com/giantswarm/azure-operator/blob/master/examples/local

## Contact

- Mailing list: [giantswarm](https://groups.google.com/forum/!forum/giantswarm)
- IRC: #[giantswarm](irc://irc.freenode.org:6667/#giantswarm) on freenode.org
- Bugs: [issues](https://github.com/giantswarm/azure-operator/issues)

## Contributing & Reporting Bugs

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches, the
contribution workflow as well as reporting bugs.

## License

azure-operator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for
details.
