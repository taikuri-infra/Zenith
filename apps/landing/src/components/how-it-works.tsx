"use client";

import { useState, useRef } from "react";
import { motion, useInView } from "framer-motion";
import { cn } from "@/lib/utils";

const tabs = [
  {
    id: "cloud",
    label: "Cloud",
    steps: [
      {
        num: "1",
        title: "Sign up",
        command: "# Visit app.freezenith.com/register",
        description:
          "Create your free account. No credit card needed. You get 1 app and 1 database instantly.",
      },
      {
        num: "2",
        title: "Push your code",
        command: "zen deploy",
        description:
          "Zenith detects your framework, builds the container, and deploys it. Zero config needed.",
      },
      {
        num: "3",
        title: "Go live",
        command: "# https://my-app.freezenith.com",
        description:
          "Your app is live with TLS, health checks, and auto-scaling. Add a custom domain anytime.",
      },
    ],
  },
  {
    id: "self-hosted",
    label: "Self-Hosted",
    steps: [
      {
        num: "1",
        title: "Install the CLI",
        command: "curl -fsSL https://get.freezenith.com | sh",
        description:
          "One command installs the Zenith CLI. Available for macOS, Linux, and WSL.",
      },
      {
        num: "2",
        title: "Deploy the platform",
        command: "zen install --provider hetzner --token hc_xxx",
        description:
          "Provisions a cluster on Hetzner Cloud. Installs the operator, API gateway, auth, and monitoring automatically.",
      },
      {
        num: "3",
        title: "Ship your apps",
        command: "cd my-app && zen deploy",
        description:
          "Push your app. Zenith detects your framework, builds the container, configures TLS, and gives you a URL.",
      },
    ],
  },
];

export function HowItWorks() {
  const [activeTab, setActiveTab] = useState("cloud");
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

  const activeSteps = tabs.find((t) => t.id === activeTab)!.steps;

  return (
    <div ref={ref}>
      {/* Tab switcher */}
      <div className="mb-12 flex justify-center">
        <div className="inline-flex rounded-full border border-border bg-surface-50/80 p-1">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                "rounded-full px-5 py-2 text-sm font-medium transition-all duration-200",
                activeTab === tab.id
                  ? "bg-accent-500 text-white shadow-lg shadow-accent-500/25"
                  : "text-neutral-400 hover:text-white"
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {/* Steps */}
      <div className="grid gap-6 md:gap-8 md:grid-cols-3">
        {activeSteps.map((step, i) => (
          <motion.div
            key={`${activeTab}-${step.num}`}
            initial={{ opacity: 0, y: 30 }}
            animate={isInView ? { opacity: 1, y: 0 } : {}}
            transition={{ duration: 0.5, delay: i * 0.12 }}
            className="relative"
          >
            {/* Step number and title */}
            <div className="mb-5 flex items-center gap-3.5">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-accent-500/10 border border-accent-500/20 text-sm font-bold text-accent-400 shrink-0">
                {step.num}
              </div>
              <h3 className="text-lg font-semibold text-white">
                {step.title}
              </h3>
            </div>

            {/* Code block */}
            <div className="rounded-xl border border-border bg-surface-50/80 p-4 font-mono text-sm overflow-x-auto">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-accent-400 select-none shrink-0">$</span>
                <span className="text-neutral-200 whitespace-nowrap">
                  {step.command}
                </span>
              </div>
            </div>

            {/* Description */}
            <p className="mt-4 text-sm text-neutral-500 leading-relaxed">
              {step.description}
            </p>

            {/* Connector line between steps (desktop only) */}
            {i < activeSteps.length - 1 && (
              <div className="absolute right-0 top-5 hidden h-px w-10 translate-x-[calc(100%-4px)] md:block">
                <div className="h-full w-full bg-gradient-to-r from-accent-500/30 to-transparent" />
              </div>
            )}
          </motion.div>
        ))}
      </div>
    </div>
  );
}
