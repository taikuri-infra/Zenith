"use client";

import { Section, SectionHeader, Reveal } from "./section";
import { Server, Building2, Check } from "lucide-react";

const options = [
  {
    icon: Server,
    tag: "Self-host",
    title: "Any Linux server",
    status: "This version",
    available: true,
    body: "One command installs FreeZenith on any server you control — a cheap VPS, bare metal, or your own cloud account. No Kubernetes, no lock-in. Docker and a domain, and you're live with automatic HTTPS.",
    bullets: ["Runs on any provider or your own hardware", "One-command install, prebuilt images — no build step", "Automatic HTTPS; your data never leaves your box"],
  },
  {
    icon: Building2,
    tag: "On-premises",
    title: "Your datacenter",
    status: "Roadmap",
    available: false,
    body: "Bring your own SAN storage and virtualization layer, and FreeZenith will run as the cloud platform on top. Data never leaves your walls — built for sovereignty.",
    bullets: ["SAN storage + virtualization", "Full data sovereignty", "Great fit for EU / regulated / banks"],
  },
];

export function Infra() {
  return (
    <Section id="infra" className="border-t border-border/50">
      <SectionHeader
        label="Bring your own infra"
        title="Your infrastructure. Your rules."
        description="FreeZenith does not host anything for you — it runs on infrastructure you provide and control. This version installs on any Linux server; managed and on-premises options are on the roadmap."
      />

      <div className="grid gap-5 md:grid-cols-2">
        {options.map((o, i) => (
          <Reveal key={o.title} delay={i * 0.1}>
            <div
              className={`glow-border group flex h-full flex-col rounded-2xl border bg-surface-50/60 p-7 transition-colors hover:border-border-hover ${
                o.available ? "border-border" : "border-border/60"
              }`}
            >
              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-accent-500/20 bg-accent-500/10 text-accent-300">
                    <o.icon className="h-5 w-5" />
                  </div>
                  <span className="font-mono text-[11px] uppercase tracking-wider text-neutral-500">
                    {o.tag}
                  </span>
                </div>
                <span
                  className={`rounded-full border px-2.5 py-1 font-mono text-[10px] uppercase tracking-wider ${
                    o.available
                      ? "border-accent-500/30 bg-accent-500/10 text-accent-300"
                      : "border-border bg-surface-100 text-neutral-500"
                  }`}
                >
                  {o.status}
                </span>
              </div>
              <h3 className="mt-5 text-xl font-semibold text-white">{o.title}</h3>
              <p className="mt-2 text-sm leading-relaxed text-neutral-400">{o.body}</p>
              <ul className="mt-5 space-y-2.5">
                {o.bullets.map((b) => (
                  <li key={b} className="flex items-center gap-2.5 text-sm text-neutral-300">
                    <Check className="h-4 w-4 shrink-0 text-accent-400" />
                    {b}
                  </li>
                ))}
              </ul>
            </div>
          </Reveal>
        ))}
      </div>
    </Section>
  );
}
