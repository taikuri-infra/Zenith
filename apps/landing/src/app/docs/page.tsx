"use client";

import { Section, SectionHeader } from "@/components/section";
import Link from "next/link";
import { motion, useInView } from "framer-motion";
import { useRef } from "react";
import {
  BookOpen,
  Terminal,
  Rocket,
  Database,
  Shield,
  Network,
  BarChart3,
  Server,
  ArrowRight,
  ExternalLink,
  Code2,
  GitBranch,
  FileCode2,
  Search,
} from "lucide-react";

const quickLinks = [
  {
    icon: Terminal,
    title: "Quick Start",
    description: "Install Zenith and deploy your first app in under 10 minutes.",
    href: "#",
    color: "accent",
  },
  {
    icon: Code2,
    title: "CLI Reference",
    description: "Complete reference for every zen command and flag.",
    href: "#",
    color: "accent",
  },
  {
    icon: FileCode2,
    title: "API Reference",
    description: "RESTful API documentation for programmatic access.",
    href: "#",
    color: "accent",
  },
  {
    icon: GitBranch,
    title: "Contributing",
    description: "How to contribute code, docs, and ideas to Zenith.",
    href: "https://github.com/DoTech/zenith/blob/main/CONTRIBUTING.md",
    color: "accent",
  },
];

const docSections = [
  {
    icon: Rocket,
    title: "Apps",
    description: "Deploy containers, configure domains, set up auto-scaling and CI/CD.",
    links: [
      { label: "Deploying Apps", href: "#" },
      { label: "Custom Domains & TLS", href: "#" },
      { label: "Environment Variables", href: "#" },
      { label: "Auto-scaling", href: "#" },
      { label: "Rollbacks", href: "#" },
    ],
  },
  {
    icon: Database,
    title: "Databases",
    description: "Provision and manage PostgreSQL, MySQL, MongoDB, and Redis instances.",
    links: [
      { label: "PostgreSQL", href: "#" },
      { label: "MySQL", href: "#" },
      { label: "MongoDB", href: "#" },
      { label: "Redis", href: "#" },
      { label: "Backups & Recovery", href: "#" },
    ],
  },
  {
    icon: Shield,
    title: "Authentication",
    description: "Configure OAuth, SAML, MFA, and per-tenant authentication realms.",
    links: [
      { label: "Auth Overview", href: "#" },
      { label: "OAuth 2.0 / OIDC", href: "#" },
      { label: "SAML", href: "#" },
      { label: "Multi-Factor Auth", href: "#" },
      { label: "Tenant Realms", href: "#" },
    ],
  },
  {
    icon: Network,
    title: "API Gateway",
    description: "Route traffic, validate JWTs, configure rate limits with Kong.",
    links: [
      { label: "Gateway Setup", href: "#" },
      { label: "Route Configuration", href: "#" },
      { label: "Rate Limiting", href: "#" },
      { label: "JWT Validation", href: "#" },
      { label: "Plugins", href: "#" },
    ],
  },
  {
    icon: BarChart3,
    title: "Monitoring",
    description: "Dashboards, alerts, and log aggregation with Grafana, Prometheus, and Loki.",
    links: [
      { label: "Dashboard Overview", href: "#" },
      { label: "Custom Alerts", href: "#" },
      { label: "Log Queries (Loki)", href: "#" },
      { label: "Metrics (Prometheus)", href: "#" },
    ],
  },
  {
    icon: Server,
    title: "Infrastructure",
    description: "Cluster management, upgrades, and Hetzner Cloud resource provisioning.",
    links: [
      { label: "Cluster Management", href: "#" },
      { label: "Node Pools", href: "#" },
      { label: "K8s Upgrades", href: "#" },
      { label: "Storage Volumes", href: "#" },
      { label: "Networking", href: "#" },
    ],
  },
];

function QuickLinkCard({
  icon: Icon,
  title,
  description,
  href,
  index,
}: {
  icon: typeof Terminal;
  title: string;
  description: string;
  href: string;
  index: number;
}) {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-50px" });

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 20 }}
      animate={isInView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.4, delay: index * 0.08 }}
    >
      <Link
        href={href}
        className="group flex flex-col rounded-2xl border border-border bg-surface-50/50 p-6 transition-all duration-300 hover:border-accent-500/20 hover:bg-surface-100/60 hover:shadow-lg hover:shadow-accent-500/5 h-full"
        {...(href.startsWith("http") ? { target: "_blank", rel: "noopener noreferrer" } : {})}
      >
        <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-accent-500/10 border border-accent-500/15 transition-all duration-300 group-hover:bg-accent-500/15 group-hover:border-accent-500/25">
          <Icon className="h-5 w-5 text-accent-400 transition-transform duration-300 group-hover:scale-110" />
        </div>
        <h3 className="mb-1.5 text-base font-semibold text-white">{title}</h3>
        <p className="text-sm text-neutral-500 leading-relaxed">{description}</p>
        <div className="mt-auto pt-4">
          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-accent-400 transition-all group-hover:gap-2">
            Read more
            <ArrowRight className="h-3 w-3" />
          </span>
        </div>
      </Link>
    </motion.div>
  );
}

function DocSectionCard({
  icon: Icon,
  title,
  description,
  links,
  index,
}: {
  icon: typeof Terminal;
  title: string;
  description: string;
  links: { label: string; href: string }[];
  index: number;
}) {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-50px" });

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 20 }}
      animate={isInView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.4, delay: index * 0.06 }}
      className="rounded-2xl border border-border bg-surface-50/50 p-6 transition-all duration-300 hover:border-border-hover"
    >
      <div className="mb-4 flex items-center gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent-500/10 border border-accent-500/15">
          <Icon className="h-4 w-4 text-accent-400" />
        </div>
        <h3 className="text-base font-semibold text-white">{title}</h3>
      </div>
      <p className="mb-5 text-sm text-neutral-500 leading-relaxed">{description}</p>
      <ul className="space-y-2">
        {links.map((link) => (
          <li key={link.label}>
            <Link
              href={link.href}
              className="group flex items-center gap-2.5 text-sm text-neutral-400 transition-colors hover:text-accent-400"
            >
              <ArrowRight className="h-3 w-3 text-neutral-600 transition-all group-hover:translate-x-0.5 group-hover:text-accent-400" />
              {link.label}
            </Link>
          </li>
        ))}
      </ul>
    </motion.div>
  );
}

export default function DocsPage() {
  return (
    <div className="pt-24">
      {/* Hero area */}
      <Section>
        <div className="text-center mb-12">
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="mb-4"
          >
            <span className="inline-flex items-center gap-2 rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 text-xs font-medium uppercase tracking-wide text-accent-400">
              <BookOpen className="h-3.5 w-3.5" />
              Documentation
            </span>
          </motion.div>
          <motion.h1
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
            className="text-3xl font-bold tracking-tight text-white md:text-4xl lg:text-5xl"
          >
            Everything you need to know
          </motion.h1>
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
            className="mx-auto mt-4 max-w-xl text-neutral-400 leading-relaxed"
          >
            Install, configure, and operate Zenith. From quick start to advanced infrastructure management.
          </motion.p>
        </div>

        {/* Search placeholder */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.3 }}
          className="mx-auto mb-16 max-w-xl"
        >
          <div className="flex items-center gap-3 rounded-xl border border-border bg-surface-50/80 backdrop-blur-sm px-4 py-3.5 transition-all hover:border-border-hover">
            <Search className="h-4 w-4 text-neutral-500" />
            <span className="text-sm text-neutral-500">
              Search documentation... (coming soon)
            </span>
            <kbd className="ml-auto hidden rounded border border-border bg-surface-200 px-2 py-0.5 text-[10px] font-mono text-neutral-500 sm:inline-block">
              Ctrl K
            </kbd>
          </div>
        </motion.div>

        {/* Quick links */}
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 mb-16">
          {quickLinks.map((link, i) => (
            <QuickLinkCard key={link.title} {...link} index={i} />
          ))}
        </div>

        {/* Divider */}
        <div className="border-t border-border" />
      </Section>

      {/* Doc sections grid */}
      <Section>
        <SectionHeader
          title="Browse by topic"
          description="Detailed guides for every Zenith feature and capability."
        />

        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
          {docSections.map((section, i) => (
            <DocSectionCard key={section.title} {...section} index={i} />
          ))}
        </div>

        {/* GitHub link */}
        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="mt-16 text-center"
        >
          <Link
            href="https://github.com/DoTech/zenith"
            className="inline-flex items-center gap-2 text-sm text-neutral-500 transition-colors hover:text-white"
            target="_blank"
            rel="noopener noreferrer"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            View the source on GitHub
          </Link>
        </motion.div>
      </Section>
    </div>
  );
}
