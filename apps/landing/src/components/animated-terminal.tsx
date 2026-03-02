"use client";

import { useEffect, useState, useRef } from "react";
import { motion } from "framer-motion";
import { cn } from "@/lib/utils";

interface TerminalLine {
  text: string;
  type: "command" | "output" | "success" | "info" | "progress";
  delay?: number;
}

const terminalSequence: TerminalLine[] = [
  { text: "zen deploy", type: "command", delay: 0 },
  { text: "", type: "output", delay: 600 },
  { text: "  Detecting framework... Next.js 15", type: "info", delay: 1000 },
  { text: "  Building container image...", type: "info", delay: 1800 },
  { text: "  [##########] Pushing image", type: "progress", delay: 3000 },
  { text: "  [##########] Configuring routing", type: "progress", delay: 4000 },
  { text: "  [##########] Provisioning TLS certificate", type: "progress", delay: 4800 },
  { text: "  [##########] Starting health checks", type: "progress", delay: 5600 },
  { text: "", type: "output", delay: 6200 },
  { text: "  Deployed! Your app is live.", type: "success", delay: 6400 },
  { text: "  URL:     https://my-app.freezenith.com", type: "success", delay: 6800 },
  { text: "  Status:  healthy (3/3 replicas)", type: "success", delay: 7100 },
  { text: "  Latency: 12ms (p99)", type: "success", delay: 7400 },
];

export function AnimatedTerminal({ className }: { className?: string }) {
  const [visibleLines, setVisibleLines] = useState<number>(0);
  const [typedCommand, setTypedCommand] = useState("");
  const [isTyping, setIsTyping] = useState(true);
  const [hasStarted, setHasStarted] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !hasStarted) {
          setHasStarted(true);
        }
      },
      { threshold: 0.3 }
    );

    if (containerRef.current) {
      observer.observe(containerRef.current);
    }

    return () => observer.disconnect();
  }, [hasStarted]);

  useEffect(() => {
    if (!hasStarted) return;

    const command = terminalSequence[0].text;
    let charIndex = 0;

    const typeInterval = setInterval(() => {
      if (charIndex <= command.length) {
        setTypedCommand(command.slice(0, charIndex));
        charIndex++;
      } else {
        clearInterval(typeInterval);
        setIsTyping(false);
        setVisibleLines(1);

        terminalSequence.forEach((line, index) => {
          if (index === 0) return;
          setTimeout(() => {
            setVisibleLines(index + 1);
          }, line.delay! - 600);
        });
      }
    }, 35);

    return () => clearInterval(typeInterval);
  }, [hasStarted]);

  const getLineColor = (type: TerminalLine["type"]) => {
    switch (type) {
      case "command": return "text-neutral-100";
      case "output": return "text-neutral-500";
      case "success": return "text-accent-400";
      case "info": return "text-neutral-400";
      case "progress": return "text-accent-400/80";
    }
  };

  return (
    <div
      ref={containerRef}
      className={cn(
        "overflow-hidden rounded-2xl border border-border bg-surface-50/80 backdrop-blur-sm",
        "glow-emerald shadow-2xl shadow-black/50",
        className
      )}
    >
      {/* Title bar */}
      <div className="flex items-center justify-between border-b border-border bg-surface-100/80 px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="flex gap-1.5">
            <div className="h-3 w-3 rounded-full bg-[#ff5f57] opacity-80 hover:opacity-100 transition-opacity" />
            <div className="h-3 w-3 rounded-full bg-[#febc2e] opacity-80 hover:opacity-100 transition-opacity" />
            <div className="h-3 w-3 rounded-full bg-[#28c840] opacity-80 hover:opacity-100 transition-opacity" />
          </div>
          <span className="ml-3 text-xs text-neutral-500 font-mono">~/my-app</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="h-1 w-1 rounded-full bg-accent-500 animate-subtle-pulse" />
          <span className="text-[10px] text-neutral-600 font-mono">zsh</span>
        </div>
      </div>

      {/* Terminal body */}
      <div className="p-5 md:p-6 font-mono text-sm leading-relaxed min-h-[280px] md:min-h-[320px]">
        {/* Command line */}
        <div className="flex items-start">
          <span className="text-accent-400 select-none shrink-0">$</span>
          <span className="ml-2 text-neutral-100">
            {typedCommand}
            {isTyping && (
              <span className="inline-block w-[2px] h-[14px] bg-accent-400 ml-[1px] align-middle cursor-blink" />
            )}
          </span>
        </div>

        {/* Output lines */}
        {terminalSequence.slice(1, visibleLines).map((line, i) => (
          <motion.div
            key={i}
            initial={{ opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
            className={cn("mt-0.5", getLineColor(line.type))}
          >
            {line.text === "" ? (
              <div className="h-3" />
            ) : (
              <span className="text-xs md:text-sm">{line.text}</span>
            )}
          </motion.div>
        ))}

        {/* Cursor after completion */}
        {visibleLines >= terminalSequence.length && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.5 }}
            className="mt-2 flex items-center"
          >
            <span className="text-accent-400 select-none">$</span>
            <span className="inline-block w-[2px] h-[14px] bg-accent-400 ml-2 cursor-blink" />
          </motion.div>
        )}
      </div>
    </div>
  );
}
