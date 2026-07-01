"use client";

import Link from "next/link";
import { useRef } from "react";
import { motion, useInView } from "framer-motion";
import { Github, Star, GitFork } from "lucide-react";
import { Section } from "./section";
import { site } from "@/lib/site";

const badges = [
  "Go",
  "k3s",
  "Kubernetes",
  "Backstage",
  "Cilium",
  "Kyverno",
  "Velero",
  "VictoriaMetrics",
  "Helm",
  "GitOps",
];

export function OpenSource() {
  const ref = useRef(null);
  const inView = useInView(ref, { once: true, margin: "-100px" });

  return (
    <Section id="open-source" className="border-t border-border/50">
      <div ref={ref} className="relative mx-auto max-w-3xl text-center">
        <motion.div
          initial={{ opacity: 0, scale: 0.85 }}
          animate={inView ? { opacity: 1, scale: 1 } : {}}
          transition={{ duration: 0.5 }}
          className="mb-8 inline-flex"
        >
          <div className="inline-flex h-16 w-16 items-center justify-center rounded-2xl border border-border bg-surface-100">
            <Github className="h-8 w-8 text-white" />
          </div>
        </motion.div>

        <motion.h2
          initial={{ opacity: 0, y: 20 }}
          animate={inView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="text-balance text-3xl font-bold text-white md:text-4xl lg:text-5xl"
        >
          Free to self-host, and yours to keep.
        </motion.h2>

        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={inView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="mx-auto mt-5 max-w-lg text-pretty leading-relaxed text-neutral-400"
        >
          Read the code, run it, fork it, and shape where it goes. FreeZenith is built in the
          open — contributions, issues and ideas welcome.
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={inView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.3 }}
          className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row"
        >
          <Link
            href={site.githubUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="group inline-flex items-center gap-2.5 rounded-xl bg-accent-500 px-6 py-3 text-sm font-semibold text-surface transition-all duration-300 hover:bg-accent-400 hover:shadow-lg hover:shadow-accent-500/25"
          >
            <Star className="h-4 w-4" />
            Star on GitHub
          </Link>
          <Link
            href={`${site.githubUrl}/fork`}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 text-sm text-neutral-400 transition-colors hover:text-white"
          >
            <GitFork className="h-4 w-4" />
            Fork the project
          </Link>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          animate={inView ? { opacity: 1 } : {}}
          transition={{ duration: 0.5, delay: 0.4 }}
          className="mt-12 flex flex-wrap items-center justify-center gap-2"
        >
          {badges.map((b) => (
            <span
              key={b}
              className="rounded-full border border-border bg-surface-100/50 px-3.5 py-1 font-mono text-xs text-neutral-500 transition-colors hover:border-border-hover hover:text-neutral-300"
            >
              {b}
            </span>
          ))}
        </motion.div>
      </div>
    </Section>
  );
}
