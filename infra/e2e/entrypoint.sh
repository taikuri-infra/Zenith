#!/usr/bin/env bash
# infra/e2e/entrypoint.sh — PID 1 of the E2E server container: make cgroup v2 nestable,
# then run dockerd in the foreground. Same trick as the official docker:dind image: on
# cgroup v2 the namespace root cgroup may not hold processes once controllers are enabled
# for children, so move everything into /init first. Without it, any inner container with
# a memory or cpu limit fails with "cannot enter cgroupv2 ... in threaded mode".
set -euo pipefail
if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
  mkdir -p /sys/fs/cgroup/init
  # Retry: processes can appear between the read and the write.
  until xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs 2>/dev/null; do :; done
  sed -e 's/ / +/g' -e 's/^/+/' < /sys/fs/cgroup/cgroup.controllers > /sys/fs/cgroup/cgroup.subtree_control
fi
# Optional Docker Hub mirror for the inner dockerd (restricted networks).
if [ -n "${ZENITH_E2E_REGISTRY_MIRROR:-}" ]; then
  /usr/local/bin/zenith-e2e-daemon-mirror /etc/docker/daemon.json "$ZENITH_E2E_REGISTRY_MIRROR"
fi
# A restarted container ("reboot") can leave a stale pid file behind.
rm -f /var/run/docker.pid /run/docker/containerd/containerd.pid
exec dockerd --host=unix:///var/run/docker.sock "$@"
