[![CircleCI](https://circleci.com/gh/giantswarm/azuretpr.svg?style=svg)](https://circleci.com/gh/giantswarm/azuretpr)

# azuretpr

Specification for Kubernetes clusters running in Giantnetes on Azure and managed
by [azure-operator](https://github.com/giantswarm/azure-operator).

## Getting Project

Clone the git repository: https://github.com/giantswarm/azuretpr.git

### How to build

Build it using the standard `go build` command.

```
go build github.com/giantswarm/azuretpr
```

However, since this project is just a specification used by the other projects,
the only goal of building is to check whether the build is successful. This is
just a library which needs to be vendored by the projects needing to use it.

## Contact

- Mailing list: [giantswarm](https://groups.google.com/forum/!forum/giantswarm)
- IRC: #[giantswarm](irc://irc.freenode.org:6667/#giantswarm) on freenode.org
- Bugs: [issues](https://github.com/giantswarm/azuretpr/issues)

## Contributing & Reporting Bugs

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches, the
contribution workflow as well as reporting bugs.

## License

azuretpr is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for
details.
