"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useInView } from "framer-motion";

type Line =
  | { kind: "cmd"; text: string }
  | { kind: "out"; text: string; tone?: "muted" | "ok" | "accent" };

const script: Line[] = [
  { kind: "cmd", text: "zen install --target hetzner" },
  { kind: "out", text: "→ provisioning k3s control plane...", tone: "muted" },
  { kind: "out", text: "→ Cilium CNI              ready", tone: "ok" },
  { kind: "out", text: "→ Kyverno policies        applied", tone: "ok" },
  { kind: "out", text: "→ Velero backups          scheduled", tone: "ok" },
  { kind: "out", text: "→ VictoriaMetrics stack   running", tone: "ok" },
  { kind: "out", text: "→ Backstage portal        deployed", tone: "ok" },
  { kind: "out", text: "✔ Your private cloud is live.", tone: "accent" },
];

const toneClass: Record<string, string> = {
  muted: "text-neutral-500",
  ok: "text-neutral-300",
  accent: "text-accent-300",
};

export function InstallTerminal() {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { once: true, margin: "-60px" });
  const [visible, setVisible] = useState(0);
  const [typed, setTyped] = useState("");

  // The first line is a command that "types" character by character.
  const firstCmd = useMemo(() => script[0].text, []);

  useEffect(() => {
    if (!inView) return;
    let i = 0;
    const typer = setInterval(() => {
      i += 1;
      setTyped(firstCmd.slice(0, i));
      if (i >= firstCmd.length) {
        clearInterval(typer);
        setVisible(1);
      }
    }, 55);
    return () => clearInterval(typer);
  }, [inView, firstCmd]);

  useEffect(() => {
    if (visible === 0 || visible >= script.length) return;
    const t = setTimeout(() => setVisible((v) => v + 1), 420);
    return () => clearTimeout(t);
  }, [visible]);

  return (
    <div ref={ref} className="glow-frame w-full rounded-2xl">
      <div className="overflow-hidden rounded-2xl border border-border bg-surface-50/90 shadow-2xl backdrop-blur">
        {/* title bar */}
        <div className="flex items-center gap-2 border-b border-border/70 bg-surface-100/80 px-4 py-3">
          <span className="h-3 w-3 rounded-full bg-[#ff5f57]" />
          <span className="h-3 w-3 rounded-full bg-[#febc2e]" />
          <span className="h-3 w-3 rounded-full bg-[#28c840]" />
          <span className="ml-3 font-mono text-xs text-neutral-500">
            ~/freezenith — self-host
          </span>
        </div>

        {/* body */}
        <div className="min-h-[280px] space-y-1.5 p-5 font-mono text-[13px] leading-relaxed sm:text-sm">
          {/* typed command */}
          <div className="flex">
            <span className="mr-2 select-none text-accent-400">$</span>
            <span className="text-white">
              {typed}
              {visible === 0 && <span className="cursor-blink text-accent-400">▋</span>}
            </span>
          </div>

          {/* streamed output */}
          {script.slice(1, visible + 1).map((line, i) => (
            <div key={i}>
              {line.kind === "cmd" ? (
                <span>
                  <span className="mr-2 select-none text-accent-400">$</span>
                  <span className="text-white">{line.text}</span>
                </span>
              ) : (
                <span className={toneClass[line.tone ?? "muted"]}>{line.text}</span>
              )}
            </div>
          ))}

          {visible >= script.length && (
            <div className="flex pt-1">
              <span className="mr-2 select-none text-accent-400">$</span>
              <span className="cursor-blink text-accent-400">▋</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
