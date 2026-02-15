import { Section, SectionHeader } from "@/components/section";
import Link from "next/link";
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
} from "lucide-react";

const docSections = [
  {
    icon: Terminal,
    title: "Getting Started",
    description: "Install Zenith and deploy your first app in under 10 minutes.",
    links: [
      { label: "Quick Start Guide", href: "#" },
      { label: "Installation", href: "#" },
      { label: "CLI Reference", href: "#" },
    ],
  },
  {
    icon: Rocket,
    title: "Apps",
    description: "Deploy containers, configure domains, set up auto-scaling and CI/CD.",
    links: [
      { label: "Deploying Apps", href: "#" },
      { label: "Custom Domains", href: "#" },
      { label: "Environment Variables", href: "#" },
      { label: "Auto-scaling", href: "#" },
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
      { label: "Log Queries", href: "#" },
    ],
  },
  {
    icon: Server,
    title: "Infrastructure",
    description: "Cluster management, upgrades, and Hetzner Cloud resource provisioning.",
    links: [
      { label: "Cluster Management", href: "#" },
      { label: "Node Pools", href: "#" },
      { label: "Upgrades", href: "#" },
      { label: "Storage Volumes", href: "#" },
    ],
  },
];

export default function DocsPage() {
  return (
    <div className="pt-20">
      <Section>
        <SectionHeader
          title="Documentation"
          description="Everything you need to install, configure, and operate Zenith."
        />

        {/* Search placeholder */}
        <div className="mx-auto mb-12 max-w-xl">
          <div className="flex items-center gap-3 rounded-lg border border-border bg-surface-50 px-4 py-3">
            <BookOpen className="h-4 w-4 text-neutral-500" />
            <span className="text-sm text-neutral-500">
              Search documentation... (coming soon)
            </span>
          </div>
        </div>

        {/* Doc sections grid */}
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {docSections.map((section) => (
            <div
              key={section.title}
              className="rounded-xl border border-border bg-surface-50 p-6 transition-all hover:border-border-hover"
            >
              <div className="mb-4 flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent-500/10 border border-accent-500/20">
                  <section.icon className="h-4 w-4 text-accent-400" />
                </div>
                <h3 className="text-base font-semibold text-white">{section.title}</h3>
              </div>
              <p className="mb-4 text-sm text-neutral-500">{section.description}</p>
              <ul className="space-y-2">
                {section.links.map((link) => (
                  <li key={link.label}>
                    <Link
                      href={link.href}
                      className="group flex items-center gap-2 text-sm text-neutral-400 transition-colors hover:text-accent-400"
                    >
                      <ArrowRight className="h-3 w-3 text-neutral-600 transition-transform group-hover:translate-x-0.5 group-hover:text-accent-400" />
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* GitHub link */}
        <div className="mt-12 text-center">
          <Link
            href="https://github.com/DoTech/zenith"
            className="inline-flex items-center gap-2 text-sm text-neutral-500 transition-colors hover:text-white"
            target="_blank"
            rel="noopener noreferrer"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            View the source on GitHub
          </Link>
        </div>
      </Section>
    </div>
  );
}
