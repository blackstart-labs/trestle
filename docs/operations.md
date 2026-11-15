# Operations and deployment

## Install with Docker Compose

From a terminal on Linux with Docker Engine and Docker Compose installed:

```bash
installer=$(curl -fsSL --proto '=https' --proto-redir '=https' https://github.com/blackstart-labs/trestle/releases/latest/download/trestle-install.sh) && bash -c "$installer"
```

The launcher downloads the bundle and its SHA-256 checksum over HTTPS. Install
a specific version by downloading `trestle-install.sh` from its GitHub release
and running `bash trestle-install.sh v1.2.3` with that release's tag. To inspect
the installer first, download the bundle and checksum, run
`sha256sum -c trestle-deploy.tar.gz.sha256`, then extract the bundle and run
`bash deploy/install.sh "$(cat image.txt)"`. The same command accepts an image
index digest from your own build in a source checkout.

The installer generates local database credentials, starts PostgreSQL, runs
migrations, waits for app readiness, and prompts for the first user and
Workspace. Rerunning the same release reuses its database and configuration.
Keep the installation directory private.

The default installation directory is `~/.local/share/trestle`, or
`$XDG_DATA_HOME/trestle` when set. From that directory:

```bash
docker compose -p trestle-production ps -a
docker compose -p trestle-production logs --tail 100 app migrate db
docker compose -p trestle-production exec app /app/trestle bootstrap
```

In external proxy mode, PostgreSQL stays on the Compose network and the
application's HTTP port binds to host loopback. Set up HTTPS with your existing
reverse proxy before signing in. For example, with an existing Caddy installation:

```caddyfile
trestle.example.com {
    reverse_proxy 127.0.0.1:3000
}
```

Point DNS at the proxy and allow its HTTPS ports. If using another proxy or a
tunnel, route to the same local upstream from the host. A proxy in a separate
container needs an explicit network route to that upstream. Configure the proxy
to set the forwarded host and scheme itself. Trestle uses secure session cookies
in production.

### Set the public URL

After HTTPS sign-in works, open **Settings → Domains** as an instance operator
and save the origin, such as `https://trestle.example.com`. This sets the base
for notification links. Notifications still deliver when the field is empty.
Changing the URL leaves the existing proxy routes and sign-in behavior in place.

A new instance assigns operator access during bootstrap. For an existing
instance, run one of these commands from the host and enter an existing username:

```bash
docker compose -p trestle-production exec app /app/trestle grantoperator
podman exec -it trestle-app /app/trestle grantoperator
```

Use `trestle grantoperator` when running the binary directly and
`trestle revokeoperator` to remove access. Reload Settings after changing a
user's access. Workspace ownership alone does not
grant access to instance settings.

## Install with rootless Podman

Use Linux with Podman 5.4 or newer, cgroup v2, and a systemd user session. Run
Podman and the installer as the account that will operate Trestle. Enable
lingering for that account with `loginctl enable-linger "$USER"` so its services
start on boot and keep running after logout.

```bash
installer=$(curl -fsSL --proto '=https' --proto-redir '=https' https://github.com/blackstart-labs/trestle/releases/latest/download/trestle-install.sh) && TRESTLE_RUNTIME=podman bash -c "$installer"
```

For a downloaded bundle, run
`TRESTLE_RUNTIME=podman bash deploy/install.sh "$(cat image.txt)"`.
The installer stores the Quadlets in
`${XDG_CONFIG_HOME:-$HOME/.config}/containers/systemd/trestle/` and credentials
in the same private data directory as the Compose installer. Quadlet starts
the database, waits for its health check, runs migrations in a one-shot service,
and waits for the app to be ready before bootstrap. PostgreSQL uses a named
volume on the Podman network. The app publishes the same loopback upstream
at `127.0.0.1:3000` in external proxy mode.

```bash
systemctl --user status trestle-app trestle-migrate trestle-db
journalctl --user -u trestle-app -u trestle-migrate -u trestle-db
podman exec -it trestle-app /app/trestle bootstrap
```

To switch between Docker and Podman, restore a verified logical backup into
a fresh deployment and cut over the proxy after checking the restored data.

## Use the bundled Caddy proxy

For a new installation, point the public hostname's DNS records at the host
and allow inbound TCP ports 80 and 443. Use Docker Compose 2.24.4 or newer, or
the rootless Podman prerequisites above with permission to bind both ports.
If the host already has a proxy on those ports, use external proxy mode.

```bash
installer=$(curl -fsSL --proto '=https' --proto-redir '=https' https://github.com/blackstart-labs/trestle/releases/latest/download/trestle-install.sh) && TRESTLE_DOMAIN=trestle.example.com bash -c "$installer"
```

Use your hostname in place of the example, and add `TRESTLE_RUNTIME=podman`
before `bash` for Podman. The same variables work with a downloaded bundle.
Caddy publishes ports 80 and 443, obtains and renews TLS certificates, and
forwards requests to Trestle over a private container network. The application
and PostgreSQL have no published ports in this mode. The reverse proxy stores
its certificates and keys in a dedicated volume.

The installer waits for Trestle readiness and starts Caddy. Then verify the
public endpoint with `curl -f https://trestle.example.com/health/ready`. If
certificate issuance or routing fails, check DNS, firewall rules and the proxy
logs:

```bash
docker compose -p trestle-production logs --tail 100 caddy
journalctl --user -u trestle-caddy
```

Sign in at the verified URL and save it in **Settings → Domains** so that
notification links use it. For now, hostname changes are made on the host.
To change it, verify DNS for the new hostname, keep a copy of the current
Caddyfile, and add the new hostname alongside the old address in the site
line. Restart Caddy and verify HTTPS and sign-in at the new address before
changing the Public URL. Retire the old address by restoring the original
Caddyfile and setting `TRESTLE_DOMAIN` in the install directory's `.env` to
the new hostname. For Compose, apply with
`docker compose -p trestle-production up -d --wait caddy` and restart Caddy
when changing only the Caddyfile. For Podman, set the same hostname in
`trestle-caddy.container` and run `systemctl --user daemon-reload` followed
by `systemctl --user restart trestle-caddy`.

## Back up and restore

From the installation directory, save a full logical database backup:

```bash
umask 077
docker compose -p trestle-production exec -T db pg_dump -U trestle -Fc trestle > trestle.dump
```

With Podman, use `podman exec trestle-db pg_dump -U trestle -Fc trestle > trestle.dump`
and back up the Quadlet directory as well as the private deployment files.

Check that the command succeeded. Store the dump and a copy of the deployment
files, including `.env`, in protected backup storage. Workspace exports omit
instance state and cannot substitute for this backup.
For a bundled HTTPS deployment, also preserve the Caddy configuration and its
certificate volume with the same protections you apply to TLS private keys.

Rehearse recovery in a separate Compose project with its own configuration and
volume. Start only its database and restore into the fresh, empty `trestle`
database:

```bash
docker compose -f compose.yaml -p trestle-recovery up -d --wait db
docker compose -f compose.yaml -p trestle-recovery exec -T db pg_restore \
  -U trestle --no-owner --exit-on-error -d trestle < trestle.dump
TRESTLE_PORT=13000 docker compose -f compose.yaml -p trestle-recovery up -d --wait app
```

Use an isolated HTTPS origin to sign in and verify your data. For disaster
recovery, first prove the restored snapshot with the image digest that created
it. Keep the original volume until the replacement has been verified.

For a Podman recovery rehearsal, use a clean test deployment. Stop
`trestle-app.service` and `trestle-migrate.service` with `systemctl --user stop`
and leave PostgreSQL running. Create a fresh database and restore into it:

```bash
podman exec trestle-db createdb -U trestle trestle_restored
podman exec -i trestle-db pg_restore -U trestle --no-owner --exit-on-error -d trestle_restored < trestle.dump
```

Point `DATABASE_URL` in the private `.env` at `trestle_restored`, keeping the
same credentials. Start `trestle-app.service` for external proxy mode or
`trestle-caddy.service` for bundled HTTPS. The Caddy unit starts the app and
its migrations as dependencies. Sign in at the test HTTPS origin.

## Upgrade or adopt an existing install

**Before changing an existing PostgreSQL container**, inspect its actual
`PGDATA` and mounts. Older examples set PostgreSQL 18's named volume to
`/var/lib/postgresql/data`. The official PostgreSQL 18 image defaults to
`/var/lib/postgresql/18/docker` and declares its volume at
`/var/lib/postgresql`. An existing database may live in an anonymous volume
attached to the running container. Take your logical backup from that running
database and restore it in isolation before changing its mounts. Preserve the
original volume. For older PostgreSQL majors or a base distribution change,
use a tested logical restore into a fresh database for adoption.

For an app upgrade on the new stack:

1. Record the running app and database image digests. Review the target release
   and save the current manifest and environment file.
2. Stop the app to pause writes, then take and verify a pre-upgrade backup.
3. Update the app image digest in `.env`, keeping its database credentials.
   Pull the app and migration images. Remove the completed migration container
   with `docker compose -p trestle-production rm -f migrate`.
4. Run `docker compose -p trestle-production up -d --wait app`, targeting
   `caddy` in place of `app` for bundled HTTPS. The migration service must exit
   successfully before the app starts. Verify sign-in through the HTTPS origin.

If migration fails, inspect the migration container's logs. A migration may
have committed earlier steps before failing. Use the verified backup and the
previous image together when the old image cannot serve the upgraded schema.

For Podman, stop the app and migration services and take a verified backup.
Pull the new app digest, update the `Image=` lines in both app and migration
Quadlets, and record it as `TRESTLE_IMAGE` in the private `.env`. Review the new
release's Quadlets for other changes, then run `systemctl --user daemon-reload`.
Start `trestle-app.service` for external proxy mode or `trestle-caddy.service`
for bundled HTTPS. Stopping the app also stops its dependent Caddy service.
Inspect the journal if the migration or application fails, and verify sign-in
through the HTTPS proxy.

## Build and operate from source

Build a local container image or publish an image index to your registry:

```bash
docker build -t trestle:local .
docker buildx build --platform linux/amd64,linux/arm64,linux/riscv64 \
  --tag registry.example.com/trestle:vX.Y.Z --push .
```

For source development, follow the [README quickstart](../README.md).
The root Compose file is the development database. The production manifest is
[`deploy/compose.yaml`](../deploy/compose.yaml).

The binary serves HTTP with no subcommand. Run `trestle migrate` explicitly
before serving and `trestle bootstrap` to initialize an empty instance.
The live probe at `/health/live` reports process availability. The ready probe
at `/health/ready` returns HTTP 503 when the database is unavailable or the
applied migration history differs from this build. Equivalent LF and CRLF
checksums are accepted for older source builds. The container's `trestle healthcheck` command
checks readiness on its local port 3000. Keep the shutdown grace period above
the application's ten-second request drain.

## Publish a release

The release workflow runs for version tags such as `v1.2.3` in the official
repository. Set repository secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`
for an account allowed to push candidates to `marufsarker/trestle`. Configure the
`production-release` GitHub environment with required reviewers and an
environment secret named `DOCKERHUB_RELEASE_TOKEN` for the same namespace.

CI builds and tests the candidate, then makes its exact image digest and
deployment bundle available on the workflow run. Before approving the
publication job, run the installer and deployment smoke test with that digest
on clean AMD64, ARM64, and RISC-V Linux hosts. The smoke test tears down its
disposable Compose project and volumes. Run it on a test host using the
matching source checkout:

```bash
bash tests/deployment-smoke.sh "$(cat /path/to/release/image.txt)"
bash tests/managed-caddy-smoke.sh "$(cat /path/to/release/image.txt)"
```

For Podman, install the unpacked candidate as a rootless user on each
supported architecture. Verify HTTPS sign-in, a reboot with persistence,
failure to start against an incompatible schema, and a recovery rehearsal
before approving publication.

The publication job assigns the version tag to the tested Docker Hub digest
and uploads the installer and bundle to the GitHub release. The hosted install
command becomes available after the first release is published.
