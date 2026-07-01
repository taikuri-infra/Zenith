"use client";

import { Section, SectionHeader, Reveal } from "./section";
import { Cloud, Building2, Check } from "lucide-react";

const options = [
  {
    icon: Cloud,
    tag: "Cloud",
    title: "Hetzner",
    body: "Bring your own Hetzner account, spin up affordable cloud or dedicated servers, and let FreeZenith provision k3s and the full platform on top. Ideal for lean teams and startups.",
    bullets: ["Use your own Hetzner account", "Cheap, powerful EU hardware", "Full platform installed for you"],
  },
  {
    icon: Building2,
    tag: "On-premises",
    title: "Your datacenter",
    body: "Bring your own SAN storage and virtualization layer; FreeZenith runs as the cloud platform on top. Data never leaves your walls — built for sovereignty.",
    bullets: ["SAN storage + virtualization", "Full data sovereignty", "Great fit for EU / regulated / banks"],
  },
];

export function Infra() {
  return (
    <Section id="infra" className="border-t border-border/50">
      <SectionHeader
        label="Bring your own infra"
        title="Your infrastructure. Your rules."
        description="FreeZenith does not host anything for you — it runs on infrastructure you provide and control. Pick the foundation that fits."
      />

      <div className="grid gap-5 md:grid-cols-2">
        {options.map((o, i) => (
          <Reveal key={o.title} delay={i * 0.1}>
            <div className="glow-border group flex h-full flex-col rounded-2xl border border-border bg-surface-50/60 p-7 transition-colors hover:border-border-hover">
              <div className="flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-accent-500/20 bg-accent-500/10 text-accent-300">
                  <o.icon className="h-5 w-5" />
                </div>
                <span className="font-mono text-[11px] uppercase tracking-wider text-neutral-500">
                  {o.tag}
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
