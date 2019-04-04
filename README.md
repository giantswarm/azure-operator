[![CircleCI](https://circleci.com/gh/giantswarm/azure-operator.svg?style=shield)](https://circleci.com/gh/giantswarm/azure-operator) [![Docker Repository on Quay](https://quay.io/repository/giantswarm/azure-operator/status "Docker Repository on Quay")](https://quay.io/repository/giantswarm/azure-operator)

# azure-operator

The azure-operator manages Kubernetes clusters running in Giantnetes on Azure.

## Getting Project

Clone the git repository: https://github.com/giantswarm/azure-operator.git

### How to build

Build it using the standard `go build` command.

```
go build github.com/giantswarm/azure-operator
```

## Running azure-operator

Create an azure-operator role using [tenant.tmpl.json](policies/tenant.tmpl.json) role definition (replace SUBSCRIPTION_ID):

```bash
az role definition create --role-definition @tenant.tmpl.json
```

If you have a service proivder you want to reuse add the azure-operator role
(replace CLIENT_ID):

```bash
az role assignment create --assignee ${CLIENT_ID} --role azure-operator
```

Otherwise create a service provider with the azure-operator role (replace
SUBSCRIPTION_ID):

```bash
export CODENAME=cluster1
az ad sp create-for-rbac -n $CODENAME-azure-operator-sp --role="azure-operator" --scopes="/subscriptions/${SUBSCRIPTION_ID}" --years 10
```

Follow [this guide][examples-local].

[examples-local]: https://github.com/giantswarm/azure-operator/blob/master/examples/README.md

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
