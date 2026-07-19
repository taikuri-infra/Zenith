"use client";

import { Section, SectionHeader, Reveal } from "./section";
import { site } from "@/lib/site";
import { Boxes, Database, Activity, ShieldCheck, LayoutGrid } from "lucide-react";

/**
 * Illustrative product preview. These are mockups of the FreeZenith portal, not a
 * live embed — the real demo is intentionally kept private for now and surfaced as
 * "coming soon". Drop real screenshots into /public and swap the mock when ready.
 */

const nav = [
  { icon: LayoutGrid, label: "Overview", active: true },
  { icon: Boxes, label: "Apps" },
  { icon: Database, label: "Databases" },
  { icon: ShieldCheck, label: "Policies" },
  { icon: Activity, label: "Metrics" },
];

const cards = [
  { label: "Running apps", value: "12", sub: "3 namespaces" },
  { label: "Databases", value: "5", sub: "backups healthy" },
  { label: "Policy pass rate", value: "100%", sub: "Kyverno" },
  { label: "Cluster CPU", value: "38%", sub: "6 nodes" },
];

export function Demo() {
  return (
    <Section id="demo" className="border-t border-border/50">
      <SectionHeader
        label="See it"
        title="One portal for the whole platform."
        description="Your team works from a Backstage-powered console — deploy, inspect and operate everything running on your private cloud."
      />

      <Reveal>
        <div className="glow-frame mx-auto max-w-5xl rounded-2xl">
          <div className="overflow-hidden rounded-2xl border border-border bg-surface-50/90 shadow-2xl">
            {/* window chrome */}
            <div className="flex items-center gap-2 border-b border-border/70 bg-surface-100/80 px-4 py-3">
              <span className="h-3 w-3 rounded-full bg-[#ff5f57]" />
              <span className="h-3 w-3 rounded-full bg-[#febc2e]" />
              <span className="h-3 w-3 rounded-full bg-[#28c840]" />
              <span className="ml-4 flex-1 truncate rounded-md bg-surface-200/70 px-3 py-1 font-mono text-[11px] text-neutral-500">
                console.your-cloud.internal
              </span>
            </div>

            {/* mock app */}
            <div className="flex min-h-[340px]">
              {/* sidebar */}
              <aside className="hidden w-52 shrink-0 border-r border-border/60 bg-surface-100/40 p-4 sm:block">
                <div className="mb-4 flex items-center gap-2 px-1">
                  <span className="h-6 w-6 rounded-md bg-gradient-to-br from-accent-400 to-accent-600" />
                  <span className="text-sm font-semibold text-white">FreeZenith</span>
                </div>
                <nav className="space-y-1">
                  {nav.map((n) => (
                    <div
                      key={n.label}
                      className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm ${
                        n.active
                          ? "bg-accent-500/10 text-accent-200"
                          : "text-neutral-400"
                      }`}
                    >
                      <n.icon className="h-4 w-4" />
                      {n.label}
                    </div>
                  ))}
                </nav>
              </aside>

              {/* content */}
              <div className="flex-1 p-5 sm:p-6">
                <div className="mb-5 flex items-center justify-between">
                  <div>
                    <h4 className="text-sm font-semibold text-white">Platform overview</h4>
                    <p className="font-mono text-[11px] text-neutral-500">production · eu-central</p>
                  </div>
                  <span className="inline-flex items-center gap-1.5 rounded-full border border-accent-500/20 bg-accent-500/10 px-3 py-1 text-[11px] font-medium text-accent-200">
                    <span className="h-1.5 w-1.5 rounded-full bg-accent-400" />
                    Healthy
                  </span>
                </div>

                <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
                  {cards.map((c) => (
                    <div key={c.label} className="rounded-xl border border-border bg-surface-100/60 p-4">
                      <p className="text-[11px] uppercase tracking-wider text-neutral-500">{c.label}</p>
                      <p className="mt-2 text-2xl font-bold text-white">{c.value}</p>
                      <p className="mt-0.5 font-mono text-[11px] text-neutral-500">{c.sub}</p>
                    </div>
                  ))}
                </div>

                {/* fake chart */}
                <div className="mt-4 rounded-xl border border-border bg-surface-100/60 p-4">
                  <div className="flex items-end gap-1.5" aria-hidden>
                    {[38, 52, 44, 61, 48, 70, 55, 63, 72, 58, 66, 80, 74, 68].map((h, i) => (
                      <span
                        key={i}
                        className="flex-1 rounded-t bg-gradient-to-t from-accent-500/30 to-accent-400/80"
                        style={{ height: `${h}px` }}
                      />
                    ))}
                  </div>
                  <p className="mt-3 font-mono text-[11px] text-neutral-500">requests / sec · last 15m</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Reveal>

      {/* demo status */}
      <Reveal delay={0.1}>
        <div className="mt-8 flex flex-col items-center gap-3">
          <span className="inline-flex items-center gap-2 rounded-full border border-border bg-surface-100 px-4 py-2 text-sm text-neutral-400">
            <span className="h-2 w-2 rounded-full bg-amber-400" />
            Live demo — <span className="font-medium text-neutral-200">{site.demo.label}</span>
          </span>
          <p className="max-w-md text-center text-xs text-neutral-600">
            Interface shown is illustrative. A public, hands-on demo is on the way.
          </p>
        </div>
      </Reveal>
    </Section>
  );
}
