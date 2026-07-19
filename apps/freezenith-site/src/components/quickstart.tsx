"use client";

import Link from "next/link";
import { Section, SectionHeader, Reveal } from "./section";
import { site } from "@/lib/site";
import { Github, ArrowRight } from "lucide-react";

const steps = [
  {
    n: "01",
    title: "Get the source",
    body: "Clone the repository and build the CLI. Everything is open — inspect it before you run it.",
    code: `git clone ${site.githubUrl}.git
cd Zenith && make cli`,
  },
  {
    n: "02",
    title: "Point at your servers",
    body: "Describe your target — a Hetzner box or on-prem hosts. FreeZenith does the rest over SSH.",
    code: `zen init
# edit zenith.yaml: hosts, storage, domain`,
  },
  {
    n: "03",
    title: "Install the platform",
    body: "One command provisions k3s and the full stack: Cilium, Kyverno, Velero, VictoriaMetrics and Backstage.",
    code: `zen install
# ✔ your private cloud is live`,
  },
];

export function QuickStart() {
  return (
    <Section id="quickstart" className="border-t border-border/50">
      <SectionHeader
        label="Self-host"
        title="From bare servers to a private cloud."
        description="Three steps. No account to create, no key to buy — just your infrastructure and the CLI."
      />

      <div className="grid gap-5 lg:grid-cols-3">
        {steps.map((s, i) => (
          <Reveal key={s.n} delay={i * 0.1}>
            <div className="flex h-full flex-col rounded-2xl border border-border bg-surface-50/60 p-6">
              <div className="flex items-center gap-3">
                <span className="font-mono text-2xl font-bold text-accent-400">{s.n}</span>
                <h3 className="text-lg font-semibold text-white">{s.title}</h3>
              </div>
              <p className="mt-3 text-sm leading-relaxed text-neutral-400">{s.body}</p>
              <pre className="mt-5 overflow-x-auto rounded-lg border border-border/70 bg-surface-100/80 p-4 font-mono text-[12.5px] leading-relaxed text-neutral-300">
                <code>{s.code}</code>
              </pre>
            </div>
          </Reveal>
        ))}
      </div>

      <Reveal delay={0.1}>
        <div className="mt-10 flex justify-center">
          <Link
            href={site.githubUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="group inline-flex items-center gap-2 rounded-xl border border-border bg-surface-100 px-6 py-3 text-sm font-medium text-neutral-200 transition-all duration-300 hover:border-border-hover hover:text-white"
          >
            <Github className="h-4 w-4" />
            Read the full guide on GitHub
            <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
          </Link>
        </div>
      </Reveal>
    </Section>
  );
}
