"use client";

import { cn } from "@/lib/utils";

interface TerminalProps {
  lines: { prompt?: string; command: string; output?: string[] }[];
  className?: string;
}

export function Terminal({ lines, className }: TerminalProps) {
  return (
    <div
      className={cn(
        "overflow-hidden rounded-xl border border-border bg-surface-50 shadow-2xl",
        "glow-emerald",
        className
      )}
    >
      {/* Title bar */}
      <div className="flex items-center gap-2 border-b border-border bg-surface-100 px-4 py-3">
        <div className="flex gap-1.5">
          <div className="h-3 w-3 rounded-full bg-red-500/60" />
          <div className="h-3 w-3 rounded-full bg-yellow-500/60" />
          <div className="h-3 w-3 rounded-full bg-green-500/60" />
        </div>
        <span className="ml-2 text-xs text-neutral-500 font-mono">terminal</span>
      </div>

      {/* Terminal body */}
      <div className="p-5 font-mono text-sm leading-relaxed">
        {lines.map((line, i) => (
          <div key={i} className="mb-1.5 last:mb-0">
            <div className="flex items-center">
              <span className="text-accent-400 select-none">{line.prompt || "$"}</span>
              <span className="ml-2 text-neutral-200">
                {i === 0 ? (
                  <span className="terminal-line inline-block">{line.command}</span>
                ) : (
                  line.command
                )}
              </span>
            </div>
            {line.output?.map((out, j) => (
              <div key={j} className="ml-0 text-neutral-500 text-xs mt-0.5">
                {out}
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
