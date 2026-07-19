/**
 * Central site configuration.
 *
 * Everything external (repo URL, demo status) is defined once here so it can be
 * swapped without touching page markup. No pricing / buy surface exists anywhere
 * in this site by design — FreeZenith is presented as a source-available platform
 * that is free to self-host. (BSL 1.1: free for internal use, no reselling; the
 * source converts to a full open-source license after the change date.)
 */

export const site = {
  name: "FreeZenith",
  tagline: "A private cloud you run yourself — source available, free to self-host.",
  description:
    "FreeZenith is a source-available internal developer platform — a full private cloud you self-host for free on Kubernetes. Bring your own infrastructure (Hetzner or on-premises) and run it on top. Licensed under BSL 1.1.",

  // Public source repository.
  githubUrl: "https://github.com/taikuri-infra/Zenith",
  license: "BSL 1.1",

  // Live demo is private for now — surfaced as "coming soon", never linked publicly.
  demo: {
    available: false,
    label: "Coming soon",
    url: "", // intentionally empty until a public demo is ready
  },
} as const;
