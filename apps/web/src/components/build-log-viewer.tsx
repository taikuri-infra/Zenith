"use client";

import { useEffect, useRef } from "react";

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

interface BuildLogViewerProps {
  entries: LogEntry[];
  streaming: boolean;
}

const levelStyles: Record<string, string> = {
  info: "text-neutral-300",
  warn: "text-amber-400",
  error: "text-red-400",
  build: "text-blue-400",
  deploy: "text-emerald-400",
};

const levelPrefix: Record<string, string> = {
  info: " INFO",
  warn: " WARN",
  error: "ERROR",
  build: "BUILD",
  deploy: "DEPLO",
};

function formatTime(iso: string): string {
  try {
    return new Date(iso).toTimeString().slice(0, 8);
  } catch {
    return "--:--:--";
  }
}

/**
 * BuildLogViewer — terminal-style log display.
 * Auto-scrolls to bottom when new entries arrive,
 * unless the user has manually scrolled up (pause mode).
 */
export function BuildLogViewer({ entries, streaming }: BuildLogViewerProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const userScrolledUp = useRef(false);

  // Track manual scroll-up to pause auto-scroll
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    const handleScroll = () => {
      const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 32;
      userScrolledUp.current = !atBottom;
    };

    el.addEventListener("scroll", handleScroll, { passive: true });
    return () => el.removeEventListener("scroll", handleScroll);
  }, []);

  // Auto-scroll when new entries arrive (unless paused)
  useEffect(() => {
    if (!userScrolledUp.current) {
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [entries]);

  if (entries.length === 0 && !streaming) {
    return (
      <div className="flex h-48 items-center justify-center rounded-lg border border-border bg-neutral-950">
        <p className="text-sm text-neutral-600">
          No log output yet — trigger a deployment to see logs here.
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-neutral-950">
      {/* Header bar */}
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <span className="font-mono text-xs font-medium text-neutral-500 uppercase tracking-wider">
          Build Output
        </span>
        <div className="flex items-center gap-2">
          {streaming && (
            <>
              <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-emerald-400" />
              <span className="text-xs text-emerald-400">Live</span>
            </>
          )}
          {!streaming && entries.length > 0 && (
            <span className="text-xs text-neutral-600">
              {entries.length} line{entries.length !== 1 ? "s" : ""}
            </span>
          )}
        </div>
      </div>

      {/* Log scroll area */}
      <div
        ref={containerRef}
        className="h-96 overflow-y-auto scroll-smooth px-4 py-3 font-mono text-xs leading-relaxed"
        style={{ scrollbarWidth: "thin" }}
      >
        {entries.map((entry, idx) => (
          <div key={idx} className="flex gap-3 py-0.5">
            {/* Timestamp */}
            <span className="shrink-0 text-neutral-600">
              [{formatTime(entry.timestamp)}]
            </span>
            {/* Level badge */}
            <span
              className={`shrink-0 font-semibold ${levelStyles[entry.level] ?? "text-neutral-400"}`}
            >
              {levelPrefix[entry.level] ?? entry.level.toUpperCase().slice(0, 5)}
            </span>
            {/* Message */}
            <span className={levelStyles[entry.level] ?? "text-neutral-300"}>
              {entry.message}
            </span>
          </div>
        ))}

        {/* Blinking cursor when streaming */}
        {streaming && (
          <div className="mt-1 flex items-center gap-1">
            <span className="text-neutral-600">[--:--:--]</span>
            <span className="h-3.5 w-2 animate-pulse bg-neutral-400 inline-block" />
          </div>
        )}

        <div ref={bottomRef} />
      </div>
    </div>
  );
}
