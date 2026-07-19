"use client";

import { Section, SectionHeader, Reveal } from "./section";
import { Check, X, Minus } from "lucide-react";

type Cell = true | false | "partial" | string;

const columns = ["FreeZenith", "Managed PaaS", "DIY Kubernetes", "Other self-hosted"];

const rows: { label: string; cells: Cell[] }[] = [
  { label: "Runs on your own infrastructure", cells: [true, false, true, true] },
  { label: "Full platform out of the box", cells: [true, "partial", false, "partial"] },
  { label: "Developer portal (Backstage)", cells: [true, false, false, false] },
  { label: "Policy, backups & metrics built in", cells: [true, "partial", false, false] },
  { label: "Data stays in your network", cells: [true, false, true, true] },
  { label: "Free to self-host", cells: [true, false, true, "partial"] },
  { label: "No vendor lock-in", cells: [true, false, true, "partial"] },
  { label: "One-command install", cells: [true, "n/a", false, "partial"] },
];

function CellValue({ value }: { value: Cell }) {
  if (value === true) return <Check className="mx-auto h-5 w-5 text-accent-400" aria-label="yes" />;
  if (value === false) return <X className="mx-auto h-5 w-5 text-neutral-600" aria-label="no" />;
  if (value === "partial")
    return <Minus className="mx-auto h-5 w-5 text-amber-500/80" aria-label="partial" />;
  return <span className="text-xs text-neutral-500">{value}</span>;
}

export function Comparison() {
  return (
    <Section id="compare" className="border-t border-border/50">
      <SectionHeader
        label="How it compares"
        title="A full platform, on infrastructure you own."
        description="Managed clouds are fast but lock you in and bill per seat. Raw Kubernetes is yours but you assemble everything by hand. FreeZenith gives you the whole platform — self-hosted, integrated, free."
      />

      <Reveal>
        <div className="overflow-x-auto rounded-2xl border border-border bg-surface-50/50">
          <table className="w-full min-w-[640px] border-collapse text-left">
            <thead>
              <tr className="border-b border-border">
                <th className="p-4 text-sm font-medium text-neutral-500" />
                {columns.map((col, i) => (
                  <th
                    key={col}
                    className={`p-4 text-center text-sm font-semibold ${
                      i === 0
                        ? "rounded-t-xl bg-accent-500/10 text-accent-300"
                        : "text-neutral-400"
                    }`}
                  >
                    {col}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, r) => (
                <tr key={row.label} className="border-b border-border/60 last:border-0">
                  <td className="p-4 text-sm text-neutral-300">{row.label}</td>
                  {row.cells.map((cell, c) => (
                    <td
                      key={c}
                      className={`p-4 text-center ${
                        c === 0 ? "bg-accent-500/[0.06]" : ""
                      } ${r === rows.length - 1 && c === 0 ? "rounded-b-xl" : ""}`}
                    >
                      <CellValue value={cell} />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Reveal>

      <Reveal delay={0.1}>
        <div className="mt-6 flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-xs text-neutral-500">
          <span className="flex items-center gap-1.5">
            <Check className="h-4 w-4 text-accent-400" /> Yes
          </span>
          <span className="flex items-center gap-1.5">
            <Minus className="h-4 w-4 text-amber-500/80" /> Partial
          </span>
          <span className="flex items-center gap-1.5">
            <X className="h-4 w-4 text-neutral-600" /> No
          </span>
        </div>
      </Reveal>
    </Section>
  );
}
