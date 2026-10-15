# Android / Termux evaluation plan

Status: unverified server target, not installation instructions or a support
promise. Access from a mobile browser is a separate future client capability.

Before publishing a supported recipe, record the Android version/device/CPU,
Termux source/version, Rust and linker support, memory/storage constraints and
available PostgreSQL version. Run the same build, migration, domain, signal and
restore tests as other supported servers. Confirm SQLx/Rustls dependencies build
on the target; a desktop Linux image is not evidence of Android compatibility.

Use app-private storage for the database and secrets; evaluate shared-storage
permissions separately for imports/exports, never put active PostgreSQL data in
an unsuitable shared filesystem. Document local/LAN exposure, TLS/proxy options,
startup credentials and backup transfer without leaking secrets.

Test foreground operation, screen-off periods, network changes, process kills,
storage pressure and recovery after interruption. Record actual CPU/RAM/battery
observations using the published workload rather than borrowing desktop results.
Document bootstrap and database/service startup only after executing them on
the stated supported environment.

Android battery policies, process restrictions and vendor behavior may suspend
or terminate services. Wake locks, battery exemptions or startup helpers do not
justify promising uninterrupted 24-hour uptime. Recommend a conventional server
when reliable unattended availability is required.
