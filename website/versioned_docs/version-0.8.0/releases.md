---
title: Releases and versions
description: How live and released docs are published.
---

# Releases and versions

The docs site has two tracks:

- `next`: generated from `website/docs/` on `main`.
- Released versions: generated snapshots under `website/versioned_docs/`.

Use the version dropdown in the navbar to switch between released docs and the
live `next` docs.

## Docs-Only Updates

Documentation changes merged to `main` publish through the Docs GitHub Actions
workflow. A docs-only update does not create a new gdproto release and does not
mint a new documentation version.

Use this for:

- Clarifying setup instructions.
- Fixing examples.
- Adding troubleshooting notes.
- Updating development documentation.

## Release Snapshots

Manual releases snapshot the current docs before tagging:

1. The release workflow computes the next semantic version.
2. Docusaurus copies `website/docs/` into
   `website/versioned_docs/version-X.Y.Z/`.
3. Docusaurus updates `website/versioned_sidebars/` and `website/versions.json`.
4. The workflow commits those generated docs files to `main`.
5. The workflow tags that commit as `vX.Y.Z`.
6. GoReleaser publishes binaries from the same tagged commit.

That keeps released docs reproducible and lets later docs-only updates continue
to publish `next` without losing old versions.

## Tag Pushes Outside The Manual Release Workflow

Tag-push releases still publish binaries through GoReleaser. They do not create
new committed docs snapshots. If maintainers create tags outside the manual
workflow, they should prepare the docs version snapshot separately before
tagging.

## Branch Protection

The manual release workflow needs permission to push one docs snapshot commit
to `main` and then push the release tag. If branch protection blocks GitHub
Actions from pushing to `main`, configure a bot exception or change the release
process to open a docs-version pull request before tagging.
