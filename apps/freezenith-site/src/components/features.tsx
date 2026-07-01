"use client";

import { Section, SectionHeader, Reveal } from "./section";
import {
  Rocket,
  Database,
  Lock,
  HardDrive,
  Network,
  BarChart3,
  GitBranch,
  Users,
  RefreshCw,
} from "lucide-react";

const features = [
  { icon: Rocket, title: "App deployments", body: "Ship containers and Git repos with zero-downtime rollouts, autoscaling and rollbacks." },
  { icon: Database, title: "Managed databases", body: "PostgreSQL and friends with automated backups and point-in-time recovery." },
  { icon: HardDrive, title: "Object storage", body: "S3-compatible buckets with presigned URLs, wired straight into your apps." },
  { icon: Network, title: "API gateway", body: "Ingress, routing, rate limiting and TLS handled at the edge of the platform." },
  { icon: Lock, title: "Security by default", body: "Kyverno guardrails and Cilium network policy enforced on every workload." },
  { icon: BarChart3, title: "Metrics & dashboards", body: "VictoriaMetrics + Grafana dashboards for apps, databases and the cluster." },
  { icon: GitBranch, title: "GitOps delivery", body: "Declarative, Git-driven deploys — your cluster state lives in version control." },
  { icon: RefreshCw, title: "Backup & restore", body: "Scheduled Velero snapshots to object storage with tested recovery." },
  { icon: Users, title: "Multi-tenancy", body: "Namespace isolation and RBAC so teams share one platform safely." },
];

export function Features() {
  return (
    <Section id="features" className="border-t border-border/50">
      <SectionHeader
        label="Features"
        title="Everything a platform team ships — built in."
        description="The capabilities you would otherwise assemble yourself, delivered as one self-hosted platform and driven from the developer portal."
      />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {features.map((f, i) => (
          <Reveal key={f.title} delay={(i % 3) * 0.06}>
            <div className="glow-border group h-full rounded-xl border border-border bg-surface-50/60 p-5 transition-colors hover:border-border-hover">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-border bg-surface-100 text-accent-300 transition-colors group-hover:border-accent-500/30">
                <f.icon className="h-5 w-5" />
              </div>
              <h3 className="mt-4 font-semibold text-white">{f.title}</h3>
              <p className="mt-1.5 text-sm leading-relaxed text-neutral-400">{f.body}</p>
            </div>
          </Reveal>
        ))}
      </div>
    </Section>
  );
}
