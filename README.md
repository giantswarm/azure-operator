[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/azure-operator/tree/master.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/azure-operator/tree/master)

# azure-operator

The azure-operator manages Kubernetes clusters running in Giantnetes on Azure.

## Branches

- `thiccc`
    - Up to and including version v2.9.0.
    - Contains all versions of legacy controllers (reconciling AzureConfig CRs) up
      to and including v2.9.0.
- `master`
    - From version v3.0.0.
    - Contains only the latest version of controllers.

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

## Pre-commit hook

- [Install pre-commit](https://pre-commit.com/#installation).
- [Install the git hook scripts](https://pre-commit.com/#3-install-the-git-hook-scripts)

Now you can [run the pre-commit against all files in the repository](https://pre-commit.com/#4-optional-run-against-all-the-files).
Or just try to make a commit and the pre-commit will be executed automatically.

## License

azure-operator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for
details.
