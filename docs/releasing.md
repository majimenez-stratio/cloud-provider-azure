# Release Process

## Pre-flight

- Identify a known good commit on the release branch
- Tag the repository with a release candidate tag and push the tag
   - An [OWNER](/OWNERS) runs `git tag -s <version>-rc.n`, inserts the changelog and pushes the tag with `git push <version>-rc.n`
- Review and discuss any potential modifications to the [support matrix/policy][support-policy]
- Review the development, getting started documentation, generation scripts, and update as necessary
- Run `make release-artifacts`, with the appropriate environment variables:
  ```
  REGISTRY="quay.io/k8s-staging" \
  PULL_POLICY="IfNotPresent" \
  MANAGER_IMAGE_TAG="<version>-rc.n" \
  make release-artifacts
  ```
- Build and push a container image to the staging repository using the release candidate tag (`<version>-rc.n`):
  ```
  REGISTRY="quay.io/k8s-staging" \
  MANAGER_IMAGE_TAG="<version>-rc.n" \
  time make docker-build

  REGISTRY="quay.io/k8s-staging" \
  MANAGER_IMAGE_TAG="<version>-rc.n" \
  time make docker-push
  ```
- Run through the "Getting Started" instructions using the artifacts generated in these steps to validate that they result in a working cluster
- Commit and PR any required changes before continuing the release process ([example](https://github.com/kubernetes-sigs/cluster-api-provider-azure/pull/249))

## (Unused) Semi-automatic

1. Make sure your repo is clean by git's standards
2. Run `go run cmd/release/main.go -remote <upstream-remote-name> -version v0.x.y`, replacing the [version][versioning], as appropriate
3. Write the [release notes](#release-notes) and make sure the binaries uploaded return the correct version
4. Push the docker images that were generated with this release tool
5. Publish release
6. [Announce][release-announcement] the release

## Manual

1. Tag the repository and push the tag
   - An [OWNER](/OWNERS) runs `git tag -s <version>` and inserts the changelog and pushes the tag with `git push <version>`
2. Create a draft release in GitHub and associate it with the tag that was just created
3. Checkout the tag and make sure git is in a clean state
4. Run `make release-artifacts`, with the appropriate environment variables:
   ```
   REGISTRY="quay.io/k8s" \
   PULL_POLICY="IfNotPresent" \
   MANAGER_IMAGE_TAG="<version>" \
   make release-artifacts
   ```
5. Attach the tarball to the drafted release
6. Attach `clusterctl` to the drafted release (for darwin and linux architectures)
7.  Write the [release notes](#release-notes) and make sure the binaries uploaded return the correct version
8.  Build and push a container image to the staging repository using the release candidate tag (`<version>`):
    ```
    REGISTRY="quay.io/k8s" \
    MANAGER_IMAGE_TAG="<version>" \
    time make docker-build

    REGISTRY="quay.io/k8s" \
    MANAGER_IMAGE_TAG="<version>" \
    time make docker-push
    ```
11. Publish release
12. [Announce][release-announcement] the release

## Versioning

cluster-api-provider-azure follows the [semantic versionining][semver] specification.

As of this writing, we have not produced as a major or minor release. 

Current pre-release versions can be expected to have breaking changes as we move towards declaring a public API version.

Example versions:
- Pre-release: `v0.1.1-alpha.1`
- Minor release: `v0.1.0`
- Patch release: `v0.1.1`
- Major release: `v1.0.0`

## Expected artifacts

1. A container image containing the azure-provider manager binary
2. A release tarball containing the manifest-templates and a script to generate the actual manifests
3. `clusterctl`
4. Release notes

## Output locations

### Container image

The container image will live in the registry `quay.io/k8s/cluster-api-provider-azure`
under the image name `cluster-api-azure-controller:<tag>` where `<tag>` is
replaced by the version being released.

### Manifests

Manifests must be generated by hand.

Running `make release-artifacts` will create a tarball that you can attach to
the drafted release.

### Binaries

The binary produced by a release is the `clusterctl` binary. There should be support for both darwin and linux architectures.

### Release Notes

Release notes are written by hand using the [release notes template][template] as a guide.

Generally, we'll make a [HackMD](https://hackmd.io/) and share the release note
 responsibility for a few days in advance of the release.

The markdown is shared in the Kubernetes slack in the channel #cluster-api-azure.

## Communication

### Minor/Patch Releases

1. Announce the release in Kubernetes Slack on the #cluster-api-azure channel.
2. An announcement email is sent to `kubernetes-sig-azure@googlegroups.com` and `kubernetes-sig-cluster-lifecycle@googlegroups.com` with the subject `[ANNOUNCE] cluster-api-provider-azure <version> has been released`

### Major Releases

1. Follow the communications process for [pre-releases](#pre-releases)
2. An announcement email is sent to `kubernetes-dev@googlegroups.com` with the subject `[ANNOUNCE] cluster-api-provider-azure <version> has been released`


[release-announcement]: #communication
[semver]: https://semver.org/#semantic-versioning-200
[support-policy]: /README.md#support-policy
[template]: /docs/release-notes-template.md
[versioning]: #versioning