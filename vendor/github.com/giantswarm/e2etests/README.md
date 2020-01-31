[![CircleCI](https://circleci.com/gh/giantswarm/e2etests.svg?&style=shield&circle-token=b073a6656900176b7e5d03d568a102b428c01afd)](https://circleci.com/gh/giantswarm/e2etests)

# e2etests
Package e2etests implements primitives for e2e tests against Giant Swarm tenant clusters.

## When to add tests

- You should add tests if they need to be executed for multiple components.
- If your test is specific to a single component add the tests in that repo.

## Test categories

- Managed Services - tests that need to be run for each Helm chart deployed in
tenant clusters.
- Provider Tests - tests that need to be run for AWS, Azure and KVM.
