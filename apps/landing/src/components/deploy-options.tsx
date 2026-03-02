"use client";

import { motion, useInView } from "framer-motion";
import { useRef } from "react";
import Link from "next/link";
import {
  Cloud,
  Server,
  Check,
  ArrowRight,
  Sparkles,
} from "lucide-react";

const options = [
  {
    icon: Cloud,
    title: "Zenith Cloud",
    subtitle: "We manage everything",
    description:
      "Sign up, push your code, and go live. No servers to manage, no infrastructure to maintain.",
    benefits: [
      "No infrastructure to manage",
      "Free tier — no credit card",
      "Auto-scaling & backups included",
      "Custom domains & TLS",
      "Upgrade anytime",
    ],
    cta: "Start Free",
    ctaHref: "https://app.freezenith.com/register",
    featured: true,
  },
  {
    icon: Server,
    title: "Self-Hosted",
    subtitle: "Full control, your infrastructure",
    description:
      "Install Zenith on your own Hetzner servers. MIT licensed, fully open-source, same platform.",
    benefits: [
      "100% open source (MIT)",
      "Your servers, your data",
      "No per-seat fees",
      "Full Kubernetes access",
      "Community support",
    ],
    cta: "Self-Host Guide",
    ctaHref: "/docs",
    featured: false,
  },
];

export function DeployOptions() {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

  return (
    <div ref={ref} className="grid gap-6 md:grid-cols-2">
      {options.map((option, i) => (
        <motion.div
          key={option.title}
          initial={{ opacity: 0, y: 30 }}
          animate={isInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: i * 0.12 }}
          className={`relative rounded-2xl border p-7 md:p-8 transition-all duration-500 ${
            option.featured
              ? "border-accent-500/30 pricing-featured shadow-lg shadow-accent-500/5"
              : "border-border bg-surface-50/50 hover:border-border-hover"
          }`}
        >
          {option.featured && (
            <div className="absolute -top-3 left-6">
              <span className="inline-flex items-center gap-1.5 rounded-full bg-accent-500 px-3 py-1 text-xs font-semibold text-white shadow-lg shadow-accent-500/25">
                <Sparkles className="h-3 w-3" />
                Recommended
              </span>
            </div>
          )}

          <div className="mb-5 flex h-11 w-11 items-center justify-center rounded-xl bg-accent-500/10 border border-accent-500/15">
            <option.icon className="h-5 w-5 text-accent-400" />
          </div>

          <h3 className="text-xl font-bold text-white">{option.title}</h3>
          <p className="mt-1 text-sm font-medium text-accent-400">
            {option.subtitle}
          </p>
          <p className="mt-3 text-sm text-neutral-400 leading-relaxed">
            {option.description}
          </p>

          <ul className="mt-6 space-y-3">
            {option.benefits.map((benefit) => (
              <li key={benefit} className="flex items-start gap-3">
                <div className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-accent-500/10">
                  <Check className="h-3 w-3 text-accent-400" />
                </div>
                <span className="text-sm text-neutral-300">{benefit}</span>
              </li>
            ))}
          </ul>

          <div className="mt-8">
            <Link
              href={option.ctaHref}
              className={`group inline-flex items-center gap-2 rounded-xl px-6 py-3 text-sm font-medium transition-all duration-300 ${
                option.featured
                  ? "bg-accent-500 text-white hover:bg-accent-600 hover:shadow-lg hover:shadow-accent-500/25"
                  : "border border-border bg-surface-200 text-neutral-300 hover:border-border-hover hover:text-white hover:bg-surface-300"
              }`}
              {...(option.ctaHref.startsWith("http")
                ? { target: "_blank", rel: "noopener noreferrer" }
                : {})}
            >
              {option.cta}
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Link>
          </div>
        </motion.div>
      ))}
    </div>
  );
}
