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
        "overflow-hidden rounded-xl border border-border bg-surface-50/80 backdrop-blur-sm",
        className
      )}
    >
      {/* Title bar */}
      <div className="flex items-center gap-2 border-b border-border bg-surface-100/80 px-4 py-3">
        <div className="flex gap-1.5">
          <div className="h-3 w-3 rounded-full bg-[#ff5f57] opacity-80" />
          <div className="h-3 w-3 rounded-full bg-[#febc2e] opacity-80" />
          <div className="h-3 w-3 rounded-full bg-[#28c840] opacity-80" />
        </div>
        <span className="ml-2 text-xs text-neutral-500 font-mono">terminal</span>
      </div>

      {/* Terminal body */}
      <div className="p-5 font-mono text-sm leading-relaxed">
        {lines.map((line, i) => (
          <div key={i} className="mb-1.5 last:mb-0">
            <div className="flex items-center">
              <span className="text-accent-400 select-none">{line.prompt || "$"}</span>
              <span className="ml-2 text-neutral-200">{line.command}</span>
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
