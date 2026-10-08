# docker: runtime image uses `alpine:latest`

The runtime stage is `FROM alpine:latest`, so builds are not reproducible and Dependabot cannot track it. Pin a version (e.g. `alpine:3.x`); then consider adding the `docker` ecosystem to `.github/dependabot.yml`, keeping `golang:<ver>` in sync with the `go` directive in `go.mod`.
