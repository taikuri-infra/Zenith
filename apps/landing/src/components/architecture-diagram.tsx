"use client";

import { motion, useInView } from "framer-motion";
import { useRef } from "react";
import {
  Globe,
  Shield,
  Database,
  BarChart3,
  Box,
  Rocket,
  Server,
  Layers,
} from "lucide-react";

function ArchNode({
  icon: Icon,
  label,
  sublabel,
  accent = false,
  delay = 0,
  isInView = false,
}: {
  icon: typeof Box;
  label: string;
  sublabel?: string;
  accent?: boolean;
  delay?: number;
  isInView?: boolean;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={isInView ? { opacity: 1, scale: 1 } : {}}
      transition={{ duration: 0.4, delay }}
      className="text-center"
    >
      <div
        className={`inline-flex items-center gap-2.5 rounded-xl border px-5 py-2.5 transition-all duration-300 hover:scale-105 ${
          accent
            ? "border-accent-500/25 bg-accent-500/8 shadow-lg shadow-accent-500/5"
            : "border-border bg-surface-100"
        }`}
      >
        <Icon
          className={`h-4 w-4 ${accent ? "text-accent-400" : "text-neutral-400"}`}
        />
        <span
          className={`text-sm font-medium ${accent ? "text-accent-300" : "text-neutral-300"}`}
        >
          {label}
        </span>
      </div>
      {sublabel && (
        <p className="mt-1.5 text-[11px] text-neutral-600">{sublabel}</p>
      )}
    </motion.div>
  );
}

function Connector({ delay = 0, isInView = false }: { delay?: number; isInView?: boolean }) {
  return (
    <motion.div
      initial={{ opacity: 0, scaleY: 0 }}
      animate={isInView ? { opacity: 1, scaleY: 1 } : {}}
      transition={{ duration: 0.3, delay }}
      className="mx-auto my-2 h-8 w-px origin-top"
    >
      <div className="h-full w-full bg-gradient-to-b from-accent-500/30 to-accent-500/10" />
      <div className="mx-auto -mt-1 h-1.5 w-1.5 rounded-full bg-accent-500/40 connector-pulse" />
    </motion.div>
  );
}

export function ArchitectureDiagram() {
  const ref = useRef(null);
  const isInView_ = useInView(ref, { once: true, margin: "-100px" });

  return (
    <div ref={ref} className="mx-auto max-w-3xl">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={isInView_ ? { opacity: 1, y: 0 } : {}}
        transition={{ duration: 0.6 }}
        className="rounded-2xl border border-border bg-surface-50/50 p-6 md:p-10"
      >
        {/* Internet */}
        <ArchNode icon={Globe} label="Internet" delay={0} isInView={isInView_} />
        <Connector delay={0.1} isInView={isInView_} />

        {/* Load Balancer */}
        <ArchNode
          icon={Server}
          label="Hetzner Load Balancer"
          sublabel="TLS termination, health checks"
          accent
          delay={0.15}
          isInView={isInView_}
        />
        <Connector delay={0.2} isInView={isInView_} />

        {/* APISIX */}
        <ArchNode
          icon={Shield}
          label="APISIX API Gateway"
          sublabel="JWT validation, rate limiting, routing"
          accent
          delay={0.25}
          isInView={isInView_}
        />
        <Connector delay={0.3} isInView={isInView_} />

        {/* Kubernetes cluster */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={isInView_ ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.35 }}
          className="rounded-xl border border-border bg-surface-100/50 p-5 md:p-7"
        >
          <div className="mb-5 flex items-center justify-center gap-2.5">
            <Layers className="h-4 w-4 text-accent-400" />
            <span className="text-sm font-semibold text-white">
              Kubernetes (k3s / CAPI)
            </span>
          </div>

          {/* Platform services grid */}
          <div className="grid grid-cols-2 gap-2.5 md:grid-cols-4">
            {[
              { icon: Box, label: "Zenith Operator", accent: true },
              { icon: Shield, label: "Auth Service", accent: false },
              { icon: BarChart3, label: "Monitoring", accent: false },
              { icon: Database, label: "DB Operators", accent: false },
            ].map((item, i) => (
              <motion.div
                key={item.label}
                initial={{ opacity: 0, scale: 0.9 }}
                animate={isInView_ ? { opacity: 1, scale: 1 } : {}}
                transition={{ duration: 0.3, delay: 0.4 + i * 0.05 }}
                className={`flex flex-col items-center gap-2 rounded-lg border p-3 text-center transition-all duration-300 hover:scale-[1.02] ${
                  item.accent
                    ? "border-accent-500/25 bg-accent-500/8"
                    : "border-border bg-surface-200/50"
                }`}
              >
                <item.icon
                  className={`h-4 w-4 ${item.accent ? "text-accent-400" : "text-neutral-500"}`}
                />
                <span
                  className={`text-[11px] font-medium leading-tight ${
                    item.accent ? "text-accent-300" : "text-neutral-400"
                  }`}
                >
                  {item.label}
                </span>
              </motion.div>
            ))}
          </div>

          {/* Connector inside cluster */}
          <div className="mx-auto my-3 h-4 w-px bg-border" />

          {/* User apps */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={isInView_ ? { opacity: 1 } : {}}
            transition={{ duration: 0.4, delay: 0.6 }}
            className="rounded-lg border border-dashed border-accent-500/25 bg-accent-500/[0.03] p-4 text-center"
          >
            <div className="flex items-center justify-center gap-2">
              <Rocket className="h-4 w-4 text-accent-400" />
              <span className="text-sm font-medium text-accent-300">
                Your Apps & Services
              </span>
            </div>
            <p className="mt-1 text-[11px] text-neutral-600">
              Containers, functions, cron jobs, workers
            </p>
          </motion.div>
        </motion.div>

        <Connector delay={0.65} isInView={isInView_} />

        {/* Hetzner */}
        <ArchNode
          icon={Server}
          label="Hetzner Cloud"
          sublabel="Servers, volumes, networking, DNS, S3 storage"
          delay={0.7}
          isInView={isInView_}
        />
      </motion.div>
    </div>
  );
}
