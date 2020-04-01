[![GoDoc](https://godoc.org/github.com/giantswarm/appcatalog?status.svg)](http://godoc.org/github.com/giantswarm/appcatalog)
[![CircleCI](https://circleci.com/gh/giantswarm/appcatalog.svg?style=shield)](https://circleci.com/gh/giantswarm/appcatalog)

# appcatalog

This repo contains a helm chart that opsctl uses to deploy `appcatalog CR`s to
our control planes.

## Crafting a release

The release process is automated but not automatic - once changes are on master,
create a github release. That will trigger circleci and publish the chart package
to the release artifacts.

### Steps:

  1. Merge your PR
  2. Draft a new release https://github.com/giantswarm/appcatalog/releases/new
  3. Update the reference to the chart in opsctl: https://github.com/giantswarm/opsctl/blob/1a4e861d0fdff530dc7bef663bb398a960191975/command/ensure/appcatalogs/command.go#L29

