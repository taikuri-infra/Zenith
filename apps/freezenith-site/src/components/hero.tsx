"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { Github, ArrowRight, Terminal } from "lucide-react";
import { site } from "@/lib/site";
import { InstallTerminal } from "./install-terminal";

const headline = [
  { text: "Your own", accent: false },
  { text: "cloud.", accent: false },
  { text: "Free to", accent: true },
  { text: "self-host.", accent: true },
];

export function Hero() {
  return (
    <section className="relative overflow-hidden pb-10 pt-32 md:pb-16 md:pt-40">
      {/* background layers */}
      <div className="absolute inset-0 grid-pattern opacity-70" />
      <div className="absolute inset-0 hero-gradient" />
      <div className="aurora-orb" />
      <div className="absolute inset-0 noise" />

      <div className="relative mx-auto max-w-6xl px-4 sm:px-6">
        <div className="flex flex-col items-center text-center">
          {/* badge */}
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="mb-8"
          >
            <span className="inline-flex items-center gap-2 rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 font-mono text-xs text-accent-300 backdrop-blur-sm">
              <span className="relative flex h-2 w-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent-400 opacity-75" />
                <span className="relative inline-flex h-2 w-2 rounded-full bg-accent-400" />
              </span>
              Source available · Free to self-host · {site.license}
            </span>
          </motion.div>

          {/* headline */}
          <h1 className="max-w-4xl text-4xl font-extrabold leading-[1.1] tracking-tight text-white sm:text-5xl md:text-6xl lg:text-7xl">
            {headline.map((word, i) => (
              <motion.span
                key={i}
                initial={{ opacity: 0, y: 28, filter: "blur(8px)" }}
                animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                transition={{ duration: 0.5, delay: 0.15 + i * 0.12, ease: [0.25, 0.4, 0.25, 1] }}
                className={`mr-[0.25em] inline-block ${word.accent ? "gradient-text-hero" : ""}`}
              >
                {word.text}
              </motion.span>
            ))}
          </h1>

          {/* subheadline */}
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.7 }}
            className="mt-6 max-w-2xl text-pretty text-base leading-relaxed text-neutral-400 sm:text-lg md:text-xl"
          >
            FreeZenith turns raw servers into a full internal developer platform — a
            private cloud on Kubernetes you host yourself. Bring your own infra on{" "}
            <span className="font-mono text-neutral-200">Hetzner</span> or{" "}
            <span className="font-mono text-neutral-200">on-premises</span>. No SaaS, no bill.
          </motion.p>

          {/* CTAs */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.85 }}
            className="mt-10 flex flex-col gap-3 sm:flex-row sm:gap-4"
          >
            <Link
              href={site.githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="group inline-flex items-center justify-center gap-2 rounded-xl bg-accent-500 px-7 py-3.5 text-sm font-semibold text-surface transition-all duration-300 hover:bg-accent-400 hover:shadow-xl hover:shadow-accent-500/25"
            >
              <Github className="h-4 w-4" />
              View on GitHub
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Link>
            <Link
              href="#quickstart"
              className="inline-flex items-center justify-center gap-2 rounded-xl border border-border bg-surface-100/60 px-7 py-3.5 text-sm font-medium text-neutral-200 backdrop-blur-sm transition-all duration-300 hover:border-border-hover hover:bg-surface-200 hover:text-white"
            >
              <Terminal className="h-4 w-4" />
              How to self-host
            </Link>
          </motion.div>

          {/* animated terminal */}
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, delay: 1.0 }}
            className="mt-16 w-full max-w-2xl"
          >
            <InstallTerminal />
          </motion.div>
        </div>
      </div>
    </section>
  );
}
