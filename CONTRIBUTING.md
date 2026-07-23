# Contributing to NetScope

NetScope uses a protected-branch workflow:

1. Branch from `dev` using `feat/`, `fix/`, `docs/`, `test/`, or `refactor/`.
2. Keep commits focused and use Conventional Commits.
3. Open a pull request into `dev`; do not push feature work directly.
4. Update tests, documentation, and `api/openapi.yaml` when behavior changes.
5. Squash-merge after required checks pass and delete the feature branch.

Releases move from `dev` to `main` only when the version is deployable.

## Local checks

```sh
make check
```

Never commit `.env`, credentials, diagnostic targets from private networks, or
captured response bodies.

