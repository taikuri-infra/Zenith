"use client";

import { Section, SectionHeader, Reveal } from "./section";
import { LayoutDashboard, Network, ShieldCheck, DatabaseBackup, LineChart } from "lucide-react";

const stack = [
  {
    icon: LayoutDashboard,
    name: "Backstage",
    role: "Developer portal",
    body: "A single pane of glass for your engineers — service catalog, software templates, docs and self-service actions. The front door to the platform.",
  },
  {
    icon: Network,
    name: "Cilium",
    role: "Networking & eBPF",
    body: "eBPF-powered CNI for pod networking, load balancing and deep network observability, with identity-aware network policy between workloads.",
  },
  {
    icon: ShieldCheck,
    name: "Kyverno",
    role: "Policy & governance",
    body: "Kubernetes-native policy engine that validates, mutates and enforces guardrails on every workload — security baselines applied automatically.",
  },
  {
    icon: DatabaseBackup,
    name: "Velero",
    role: "Backup & disaster recovery",
    body: "Scheduled cluster and volume backups to object storage, with tested restores — so your private cloud survives a bad day.",
  },
  {
    icon: LineChart,
    name: "VictoriaMetrics",
    role: "Metrics & observability",
    body: "A fast, resource-efficient metrics stack for long-term time-series storage, dashboards and alerting across the whole platform.",
  },
];

export function Stack() {
  return (
    <Section id="stack" className="border-t border-border/50">
      <SectionHeader
        label="The stack"
        title="Best-in-class open source, pre-integrated."
        description="FreeZenith wires proven CNCF-ecosystem projects into one coherent platform on k3s / Kubernetes — so you get the whole thing working together, out of the box."
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {stack.map((s, i) => (
          <Reveal key={s.name} delay={i * 0.06}>
            <div className="glow-border group h-full rounded-xl border border-border bg-surface-50/60 p-6 transition-colors hover:border-border-hover">
              <div className="flex items-center justify-between">
                <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-accent-500/20 bg-accent-500/10 text-accent-300">
                  <s.icon className="h-5 w-5" />
                </div>
                <span className="font-mono text-[11px] uppercase tracking-wider text-neutral-500">
                  {s.role}
                </span>
              </div>
              <h3 className="mt-5 text-lg font-semibold text-white">{s.name}</h3>
              <p className="mt-2 text-sm leading-relaxed text-neutral-400">{s.body}</p>
            </div>
          </Reveal>
        ))}

        {/* foundation card */}
        <Reveal delay={0.3}>
          <div className="flex h-full flex-col justify-center rounded-xl border border-accent-500/20 bg-gradient-to-br from-accent-950/40 to-surface-50/60 p-6">
            <span className="font-mono text-[11px] uppercase tracking-wider text-accent-300">
              Foundation
            </span>
            <h3 className="mt-2 text-lg font-semibold text-white">k3s / Kubernetes</h3>
            <p className="mt-2 text-sm leading-relaxed text-neutral-400">
              A lightweight, certified Kubernetes distribution as the base layer — small
              enough for a single node, ready to scale to a cluster.
            </p>
          </div>
        </Reveal>
      </div>
    </Section>
  );
}
