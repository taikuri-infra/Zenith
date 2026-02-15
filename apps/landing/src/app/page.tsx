"use client";

import Link from "next/link";
import { motion, useInView } from "framer-motion";
import { useRef } from "react";
import { Section, SectionHeader } from "@/components/section";
import { FeatureCard } from "@/components/feature-card";
import { AnimatedTerminal } from "@/components/animated-terminal";
import { TrustBar } from "@/components/trust-bar";
import { ArchitectureDiagram } from "@/components/architecture-diagram";
import { PricingComparison } from "@/components/pricing-comparison";
import {
  Rocket,
  Database,
  Shield,
  HardDrive,
  Network,
  BarChart3,
  ArrowRight,
  Github,
  Star,
  Users,
  ChevronRight,
  Copy,
  Check,
} from "lucide-react";
import { useState } from "react";

export default function LandingPage() {
  return (
    <div className="relative">
      {/* ===== HERO SECTION ===== */}
      <HeroSection />

      {/* ===== TRUST BAR ===== */}
      <TrustBar />

      {/* ===== FEATURES SECTION ===== */}
      <Section id="features">
        <SectionHeader
          label="Features"
          title="Everything you need. Built in."
          description="Zenith ships with a complete platform out of the box. No plugins to install, no third-party services to manage."
        />

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <FeatureCard
            icon={Rocket}
            title="Apps"
            description="Deploy any container with zero-downtime deploys, auto-scaling, rollbacks, and custom domains. From Docker images or Git repos."
            index={0}
          />
          <FeatureCard
            icon={Database}
            title="Databases"
            description="PostgreSQL, MySQL, MongoDB, Redis -- all managed. Automated backups, point-in-time recovery, connection pooling included."
            index={1}
          />
          <FeatureCard
            icon={Shield}
            title="Auth"
            description="Built-in authentication and authorization. OAuth 2.0, SAML, MFA, per-tenant realms. No external Keycloak needed."
            index={2}
          />
          <FeatureCard
            icon={HardDrive}
            title="Storage"
            description="S3-compatible object storage integrated with Hetzner. Buckets, presigned URLs, CDN-ready. Seamlessly connected to your apps."
            index={3}
          />
          <FeatureCard
            icon={Network}
            title="API Gateway"
            description="Kong-powered API gateway with rate limiting, JWT validation, CORS, and request transformation. CRD-driven configuration."
            index={4}
          />
          <FeatureCard
            icon={BarChart3}
            title="Monitoring"
            description="Grafana, Prometheus, and Loki out of the box. Pre-built dashboards for apps, databases, and infrastructure."
            index={5}
          />
        </div>
      </Section>

      {/* ===== HOW IT WORKS SECTION ===== */}
      <HowItWorksSection />

      {/* ===== ARCHITECTURE SECTION ===== */}
      <Section id="architecture" className="border-t border-border/50">
        <SectionHeader
          label="Architecture"
          title="Built on proven technology"
          description="Zenith combines the best open-source tools into a unified platform managed by a Kubernetes operator."
        />
        <ArchitectureDiagram />
      </Section>

      {/* ===== PRICING COMPARISON SECTION ===== */}
      <Section id="pricing" className="border-t border-border/50">
        <SectionHeader
          label="Pricing"
          title="10x cheaper. Same power."
          description="Run the same workloads for a fraction of the cost. Zenith is free -- you only pay for Hetzner infrastructure."
        />
        <PricingComparison />
      </Section>

      {/* ===== OPEN SOURCE SECTION ===== */}
      <OpenSourceSection />

      {/* ===== CTA FOOTER SECTION ===== */}
      <CTASection />
    </div>
  );
}

/* ===== Hero Section ===== */
function HeroSection() {
  return (
    <section className="relative overflow-hidden pt-32 pb-8 md:pt-40 md:pb-12">
      {/* Background layers */}
      <div className="absolute inset-0 grid-pattern opacity-60" />
      <div className="absolute inset-0 hero-gradient" />
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] bg-accent-500/[0.04] rounded-full blur-[120px]" />
      <div className="absolute inset-0 noise" />

      <div className="relative mx-auto max-w-6xl px-4 sm:px-6">
        <div className="flex flex-col items-center text-center">
          {/* Badge */}
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="mb-8"
          >
            <span className="inline-flex items-center gap-2 rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 text-sm text-accent-400 backdrop-blur-sm">
              <span className="relative flex h-2 w-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent-400 opacity-75" />
                <span className="relative inline-flex h-2 w-2 rounded-full bg-accent-400" />
              </span>
              100% Free and Open Source
            </span>
          </motion.div>

          {/* Headline */}
          <motion.h1
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.1 }}
            className="max-w-4xl text-4xl font-extrabold tracking-tight text-white sm:text-5xl md:text-6xl lg:text-7xl leading-[1.1]"
          >
            Your Own Cloud Platform.{" "}
            <span className="gradient-text-hero">10x Cheaper.</span>
          </motion.h1>

          {/* Subheadline */}
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="mt-6 max-w-2xl text-base text-neutral-400 sm:text-lg md:text-xl leading-relaxed"
          >
            One <code className="rounded bg-surface-200 px-1.5 py-0.5 font-mono text-sm text-accent-400">zen install</code> command
            turns Hetzner Cloud into your own platform -- apps, databases, auth, storage, gateway, monitoring.
          </motion.p>

          {/* CTA Buttons */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.3 }}
            className="mt-10 flex flex-col gap-3 sm:flex-row sm:gap-4"
          >
            <Link
              href="#get-started"
              className="group inline-flex items-center justify-center gap-2 rounded-xl bg-accent-500 px-7 py-3.5 text-sm font-semibold text-white transition-all duration-300 hover:bg-accent-600 hover:shadow-xl hover:shadow-accent-500/25 hover:scale-[1.02]"
            >
              Get Started
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Link>
            <Link
              href="https://github.com/DoTech/zenith"
              className="inline-flex items-center justify-center gap-2 rounded-xl border border-border bg-surface-50/50 px-7 py-3.5 text-sm font-medium text-neutral-300 backdrop-blur-sm transition-all duration-300 hover:border-border-hover hover:text-white hover:bg-surface-100"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Github className="h-4 w-4" />
              View on GitHub
            </Link>
          </motion.div>

          {/* Animated Terminal */}
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, delay: 0.5 }}
            className="mt-16 w-full max-w-2xl"
          >
            <AnimatedTerminal />
          </motion.div>
        </div>
      </div>
    </section>
  );
}

/* ===== How It Works Section ===== */
function HowItWorksSection() {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

  const steps = [
    {
      num: "1",
      title: "Install the CLI",
      command: "curl -fsSL https://get.freezenith.com | sh",
      description:
        "One command installs the Zenith CLI. Available for macOS, Linux, and WSL. Homebrew and apt packages also available.",
    },
    {
      num: "2",
      title: "Deploy your platform",
      command: "zen install --provider hetzner --token hc_xxx",
      description:
        "Provisions a management cluster on Hetzner Cloud. Installs the Zenith operator, API gateway, auth service, and monitoring stack automatically.",
    },
    {
      num: "3",
      title: "Ship your apps",
      command: "cd my-app && zen deploy",
      description:
        "Push your app. Zenith detects your framework, builds the container, configures TLS, and gives you a URL. Zero config needed.",
    },
  ];

  return (
    <Section id="how-it-works" className="border-t border-border/50">
      <SectionHeader
        label="How it Works"
        title="Three steps. Five minutes."
        description="From zero to a fully operational cloud platform. No DevOps degree required."
      />

      <div ref={ref} className="grid gap-6 md:gap-8 md:grid-cols-3">
        {steps.map((step, i) => (
          <motion.div
            key={step.num}
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
              <h3 className="text-lg font-semibold text-white">{step.title}</h3>
            </div>

            {/* Code block */}
            <div className="rounded-xl border border-border bg-surface-50/80 p-4 font-mono text-sm overflow-x-auto">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-accent-400 select-none shrink-0">$</span>
                <span className="text-neutral-200 whitespace-nowrap">{step.command}</span>
              </div>
            </div>

            {/* Description */}
            <p className="mt-4 text-sm text-neutral-500 leading-relaxed">
              {step.description}
            </p>

            {/* Connector line between steps (desktop only) */}
            {i < steps.length - 1 && (
              <div className="absolute right-0 top-5 hidden h-px w-10 translate-x-[calc(100%-4px)] md:block">
                <div className="h-full w-full bg-gradient-to-r from-accent-500/30 to-transparent" />
              </div>
            )}
          </motion.div>
        ))}
      </div>
    </Section>
  );
}

/* ===== Open Source Section ===== */
function OpenSourceSection() {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

  return (
    <Section className="border-t border-border/50">
      <div ref={ref} className="mx-auto max-w-3xl text-center">
        {/* GitHub icon */}
        <motion.div
          initial={{ opacity: 0, scale: 0.8 }}
          animate={isInView ? { opacity: 1, scale: 1 } : {}}
          transition={{ duration: 0.5 }}
          className="mb-8"
        >
          <div className="inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-surface-100 border border-border">
            <Github className="h-8 w-8 text-white" />
          </div>
        </motion.div>

        <motion.h2
          initial={{ opacity: 0, y: 20 }}
          animate={isInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="text-3xl font-bold text-white md:text-4xl lg:text-5xl"
        >
          100% Free, Forever.
        </motion.h2>

        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={isInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="mx-auto mt-5 max-w-lg text-neutral-400 leading-relaxed"
        >
          MIT licensed. No vendor lock-in. No hidden features. Fork it, modify it,
          self-host it. The entire platform is yours.
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={isInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.3 }}
          className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row sm:gap-6"
        >
          <Link
            href="https://github.com/DoTech/zenith"
            className="group inline-flex items-center gap-2.5 rounded-xl bg-surface-200 border border-border px-6 py-3 text-sm font-medium text-white transition-all duration-300 hover:border-border-hover hover:bg-surface-300 hover:shadow-lg"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Star className="h-4 w-4 text-yellow-500" />
            Star on GitHub
            <ChevronRight className="h-3.5 w-3.5 text-neutral-500 transition-transform group-hover:translate-x-0.5" />
          </Link>
          <Link
            href="https://github.com/DoTech/zenith/blob/main/CONTRIBUTING.md"
            className="inline-flex items-center gap-2 text-sm text-neutral-400 transition-colors hover:text-white"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Users className="h-4 w-4" />
            Become a contributor
          </Link>
        </motion.div>

        {/* Tech stack badges */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={isInView ? { opacity: 1 } : {}}
          transition={{ duration: 0.5, delay: 0.4 }}
          className="mt-12 flex flex-wrap items-center justify-center gap-2"
        >
          {[
            "Go",
            "Kubernetes",
            "Next.js",
            "TypeScript",
            "Helm",
            "Kong",
            "Grafana",
            "PostgreSQL",
          ].map((tech) => (
            <span
              key={tech}
              className="rounded-full border border-border bg-surface-100/50 px-3.5 py-1 text-xs text-neutral-500 transition-colors hover:text-neutral-300 hover:border-border-hover"
            >
              {tech}
            </span>
          ))}
        </motion.div>
      </div>
    </Section>
  );
}

/* ===== CTA Section ===== */
function CTASection() {
  const [copied, setCopied] = useState(false);
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

  const installCommand = "zen install --provider hetzner --token hc_xxx";

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(installCommand);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // fallback silently
    }
  };

  return (
    <Section id="get-started" className="border-t border-border/50">
      <motion.div
        ref={ref}
        initial={{ opacity: 0, y: 20 }}
        animate={isInView ? { opacity: 1, y: 0 } : {}}
        transition={{ duration: 0.6 }}
        className="relative overflow-hidden rounded-2xl border border-accent-500/15 bg-gradient-to-br from-accent-950/40 via-surface-50 to-surface-50 p-8 md:p-14 text-center"
      >
        {/* Background glow */}
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[500px] h-[250px] bg-accent-500/8 rounded-full blur-[100px]" />
        <div className="absolute inset-0 noise" />

        <div className="relative">
          <h2 className="text-3xl font-bold text-white md:text-4xl lg:text-5xl">
            Deploy your first app in 5 minutes
          </h2>
          <p className="mx-auto mt-5 max-w-lg text-neutral-400 leading-relaxed">
            No credit card. No sign-up. Just your Hetzner account and one command.
          </p>

          {/* Install command with copy */}
          <div className="mx-auto mt-10 max-w-lg">
            <div className="group flex items-center rounded-xl border border-border bg-surface-100/80 backdrop-blur-sm px-5 py-3.5 font-mono text-sm transition-all hover:border-border-hover">
              <span className="text-accent-400 select-none">$</span>
              <span className="ml-2 flex-1 text-left text-neutral-200 overflow-x-auto whitespace-nowrap scrollbar-hide">
                {installCommand}
              </span>
              <button
                onClick={handleCopy}
                className="ml-3 shrink-0 rounded-lg p-1.5 text-neutral-500 transition-all hover:text-white hover:bg-surface-300"
                aria-label="Copy to clipboard"
              >
                {copied ? (
                  <Check className="h-4 w-4 text-accent-400" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </button>
            </div>
          </div>

          <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row sm:gap-4">
            <Link
              href="/docs"
              className="group inline-flex items-center gap-2 rounded-xl bg-accent-500 px-7 py-3.5 text-sm font-semibold text-white transition-all duration-300 hover:bg-accent-600 hover:shadow-xl hover:shadow-accent-500/25"
            >
              Read the Docs
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Link>
            <Link
              href="https://github.com/DoTech/zenith"
              className="inline-flex items-center gap-2 text-sm text-neutral-400 transition-colors hover:text-white"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Github className="h-4 w-4" />
              View source on GitHub
            </Link>
          </div>
        </div>
      </motion.div>
    </Section>
  );
}
