"use client";

import Link from "next/link";
import { motion, useInView } from "framer-motion";
import { useRef } from "react";
import { Section, SectionHeader } from "@/components/section";
import { FeatureCard } from "@/components/feature-card";
import { AnimatedTerminal } from "@/components/animated-terminal";
import { TrustBar } from "@/components/trust-bar";
import { DeployOptions } from "@/components/deploy-options";
import { HowItWorks } from "@/components/how-it-works";
import { PricingTabs } from "@/components/pricing-tabs";
import { ArchitectureDiagram } from "@/components/architecture-diagram";
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
} from "lucide-react";

export default function LandingPage() {
  return (
    <div className="relative">
      {/* ===== HERO SECTION ===== */}
      <HeroSection />

      {/* ===== TRUST BAR ===== */}
      <TrustBar />

      {/* ===== DEPLOY OPTIONS SECTION ===== */}
      <Section id="deploy" className="border-t border-border/50">
        <SectionHeader
          label="Deploy"
          title="Cloud or Self-Hosted. Same platform."
          description="Use Zenith Cloud for zero-ops deployment, or self-host the open-source platform on your own infrastructure."
        />
        <DeployOptions />
      </Section>

      {/* ===== FEATURES SECTION ===== */}
      <Section id="features" className="border-t border-border/50">
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
            description="PostgreSQL, MySQL, MongoDB, Redis — all managed. Automated backups, point-in-time recovery, connection pooling included."
            index={1}
          />
          <FeatureCard
            icon={Shield}
            title="Auth"
            description="Built-in authentication and authorization. OAuth 2.0, SAML, MFA, per-tenant realms. No external identity provider needed."
            index={2}
          />
          <FeatureCard
            icon={HardDrive}
            title="Storage"
            description="S3-compatible object storage. Buckets, presigned URLs, CDN-ready. Seamlessly connected to your apps."
            index={3}
          />
          <FeatureCard
            icon={Network}
            title="API Gateway"
            description="APISIX-powered API gateway with rate limiting, JWT validation, CORS, and request transformation. CRD-driven configuration."
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
      <Section id="how-it-works" className="border-t border-border/50">
        <SectionHeader
          label="How it Works"
          title="Three steps. Five minutes."
          description="From zero to a live app. Pick your path."
        />
        <HowItWorks />
      </Section>

      {/* ===== PRICING SECTION ===== */}
      <Section id="pricing" className="border-t border-border/50">
        <SectionHeader
          label="Pricing"
          title="Start free. Scale when you grow."
          description="No credit card required. Upgrade when your project takes off."
        />
        <PricingTabs />
      </Section>

      {/* ===== ARCHITECTURE SECTION ===== */}
      <Section id="architecture" className="border-t border-border/50">
        <SectionHeader
          label="Architecture"
          title="Built on proven technology"
          description="Zenith combines the best open-source tools into a unified platform managed by a Kubernetes operator."
        />
        <ArchitectureDiagram />
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
              Free tier — no credit card required
            </span>
          </motion.div>

          {/* Headline */}
          <motion.h1
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.1 }}
            className="max-w-4xl text-4xl font-extrabold tracking-tight text-white sm:text-5xl md:text-6xl lg:text-7xl leading-[1.1]"
          >
            Ship Faster.{" "}
            <span className="gradient-text-hero">Scale Freely.</span>
          </motion.h1>

          {/* Subheadline */}
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="mt-6 max-w-2xl text-base text-neutral-400 sm:text-lg md:text-xl leading-relaxed"
          >
            Deploy apps, databases, and APIs on Zenith Cloud in seconds — or
            self-host on your own infrastructure.
          </motion.p>

          {/* CTA Buttons */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.3 }}
            className="mt-10 flex flex-col gap-3 sm:flex-row sm:gap-4"
          >
            <Link
              href="https://app.freezenith.com/register"
              className="group inline-flex items-center justify-center gap-2 rounded-xl bg-accent-500 px-7 py-3.5 text-sm font-semibold text-white transition-all duration-300 hover:bg-accent-600 hover:shadow-xl hover:shadow-accent-500/25 hover:scale-[1.02]"
            >
              Start Free
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Link>
            <Link
              href="/docs"
              className="inline-flex items-center justify-center gap-2 rounded-xl border border-border bg-surface-50/50 px-7 py-3.5 text-sm font-medium text-neutral-300 backdrop-blur-sm transition-all duration-300 hover:border-border-hover hover:text-white hover:bg-surface-100"
            >
              Self-Host Guide
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
          Open Source. MIT Licensed.
        </motion.h2>

        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={isInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="mx-auto mt-5 max-w-lg text-neutral-400 leading-relaxed"
        >
          Fork it, modify it, self-host it — or use Zenith Cloud and let us
          handle everything. The entire platform is open source.
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
            "APISIX",
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
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-100px" });

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
            Ready to deploy?
          </h2>
          <p className="mx-auto mt-5 max-w-lg text-neutral-400 leading-relaxed">
            Start with the free tier — no credit card, no setup. Or self-host on
            your own infrastructure.
          </p>

          <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row sm:gap-4">
            <Link
              href="https://app.freezenith.com/register"
              className="group inline-flex items-center gap-2 rounded-xl bg-accent-500 px-7 py-3.5 text-sm font-semibold text-white transition-all duration-300 hover:bg-accent-600 hover:shadow-xl hover:shadow-accent-500/25"
            >
              Start Free on Cloud
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Link>
            <Link
              href="/docs"
              className="inline-flex items-center gap-2 text-sm text-neutral-400 transition-colors hover:text-white"
            >
              Self-Host Guide
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </div>
      </motion.div>
    </Section>
  );
}
