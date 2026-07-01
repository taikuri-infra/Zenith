"use client";

import { Section, Reveal } from "./section";
import { Server, Boxes, GitBranch } from "lucide-react";

const points = [
  {
    icon: Server,
    title: "Runs on your hardware",
    body: "Point FreeZenith at fresh servers — a Hetzner box or on-prem hosts with SAN storage and virtualization — and it installs a production k3s cluster on top.",
  },
  {
    icon: Boxes,
    title: "A full platform, not a toolbox",
    body: "Networking, security policy, backups, metrics and a developer portal come pre-integrated. No stitching together a dozen Helm charts by hand.",
  },
  {
    icon: GitBranch,
    title: "Self-service for your team",
    body: "Developers ship apps, databases and services through a Backstage portal and GitOps — while you keep full control of the underlying cluster.",
  },
];

export function WhatIs() {
  return (
    <Section id="what" className="border-t border-border/50">
      <div className="grid items-start gap-12 lg:grid-cols-2 lg:gap-16">
        <Reveal>
          <span className="mb-4 inline-block rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 font-mono text-xs font-medium uppercase tracking-wider text-accent-300">
            What it is
          </span>
          <h2 className="text-balance text-3xl font-bold tracking-tight text-white md:text-4xl lg:text-5xl">
            An internal developer platform you actually own.
          </h2>
          <p className="mt-5 text-pretty text-base leading-relaxed text-neutral-400 md:text-lg">
            FreeZenith is a source-available, free-to-self-host{" "}
            <span className="text-neutral-200">internal developer platform</span> — the
            private-cloud layer that turns bare Kubernetes into a self-service platform for
            your engineers. Everything runs on infrastructure you control, so there is no
            SaaS dependency, no per-seat pricing, and nothing leaves your network.
          </p>
        </Reveal>

        <div className="flex flex-col gap-4">
          {points.map((p, i) => (
            <Reveal key={p.title} delay={i * 0.08}>
              <div className="glow-border group flex gap-4 rounded-xl border border-border bg-surface-50/60 p-5 transition-colors hover:border-border-hover">
                <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg border border-accent-500/20 bg-accent-500/10 text-accent-300">
                  <p.icon className="h-5 w-5" />
                </div>
                <div>
                  <h3 className="font-semibold text-white">{p.title}</h3>
                  <p className="mt-1.5 text-sm leading-relaxed text-neutral-400">{p.body}</p>
                </div>
              </div>
            </Reveal>
          ))}
        </div>
      </div>
    </Section>
  );
}
