import Link from "next/link";
import { Section, SectionHeader } from "@/components/section";
import { FeatureCard } from "@/components/feature-card";
import { Terminal } from "@/components/terminal";
import { PricingCard } from "@/components/pricing-card";
import {
  Rocket,
  Database,
  Shield,
  HardDrive,
  Network,
  BarChart3,
  Terminal as TerminalIcon,
  Upload,
  Scaling,
  ArrowRight,
  Github,
  Star,
  Users,
  Zap,
  Server,
  Layers,
  Box,
  ChevronRight,
} from "lucide-react";

export default function LandingPage() {
  return (
    <div className="relative">
      {/* ===== HERO SECTION ===== */}
      <section className="relative overflow-hidden pt-28 pb-20 md:pt-36 md:pb-28">
        {/* Background effects */}
        <div className="absolute inset-0 grid-pattern" />
        <div className="absolute inset-0 hero-gradient" />
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-accent-500/5 rounded-full blur-3xl" />

        <div className="relative mx-auto max-w-6xl px-4 sm:px-6">
          <div className="flex flex-col items-center text-center">
            {/* Badge */}
            <div className="mb-6 animate-fade-in">
              <span className="inline-flex items-center gap-2 rounded-full border border-accent-500/30 bg-accent-500/10 px-4 py-1.5 text-sm text-accent-400">
                <Zap className="h-3.5 w-3.5" />
                100% Free and Open Source
              </span>
            </div>

            {/* Headline */}
            <h1 className="animate-fade-in-up max-w-4xl text-4xl font-extrabold tracking-tight text-white sm:text-5xl md:text-6xl lg:text-7xl">
              Your own cloud.{" "}
              <span className="gradient-text">Zero complexity.</span>
            </h1>

            {/* Subheadline */}
            <p className="mt-6 max-w-2xl animate-fade-in-up text-base text-neutral-400 opacity-0 delay-200 sm:text-lg md:text-xl">
              100% free, open-source Kubernetes PaaS on Hetzner Cloud.
              One command to deploy apps, databases, auth, and everything you need.
            </p>

            {/* CTA Buttons */}
            <div className="mt-8 flex flex-col gap-3 animate-fade-in-up opacity-0 delay-300 sm:flex-row sm:gap-4">
              <Link
                href="#get-started"
                className="group inline-flex items-center justify-center gap-2 rounded-lg bg-accent-500 px-6 py-3 text-sm font-semibold text-white transition-all hover:bg-accent-600 hover:shadow-lg hover:shadow-accent-500/25"
              >
                Get Started
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </Link>
              <Link
                href="https://github.com/DoTech/zenith"
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-border bg-surface-50 px-6 py-3 text-sm font-medium text-neutral-300 transition-all hover:border-border-hover hover:text-white"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Github className="h-4 w-4" />
                View on GitHub
              </Link>
            </div>

            {/* Terminal */}
            <div className="mt-12 w-full max-w-2xl animate-fade-in-up opacity-0 delay-400">
              <Terminal
                lines={[
                  {
                    command: "zen install --provider hetzner --token hc_xxx",
                    output: [
                      "  Provisioning management cluster on Hetzner Cloud...",
                      "  Installing Zenith operator, gateway, monitoring...",
                      "  Ready! Dashboard: https://zenith.your-domain.com",
                    ],
                  },
                ]}
              />
            </div>
          </div>
        </div>
      </section>

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
          />
          <FeatureCard
            icon={Database}
            title="Databases"
            description="PostgreSQL, MySQL, MongoDB, Redis -- all managed. Automated backups, point-in-time recovery, connection pooling included."
          />
          <FeatureCard
            icon={Shield}
            title="Auth"
            description="Built-in authentication and authorization. OAuth 2.0, SAML, MFA, per-tenant realms. No external Keycloak needed."
          />
          <FeatureCard
            icon={HardDrive}
            title="Storage"
            description="S3-compatible object storage integrated with Hetzner. Buckets, presigned URLs, CDN-ready. Seamlessly connected to your apps."
          />
          <FeatureCard
            icon={Network}
            title="API Gateway"
            description="Kong-powered API gateway with rate limiting, JWT validation, CORS, and request transformation. CRD-driven configuration."
          />
          <FeatureCard
            icon={BarChart3}
            title="Monitoring"
            description="Grafana, Prometheus, and Loki out of the box. Pre-built dashboards for apps, databases, and infrastructure."
          />
        </div>
      </Section>

      {/* ===== HOW IT WORKS SECTION ===== */}
      <Section id="how-it-works" className="border-t border-border">
        <SectionHeader
          label="How it Works"
          title="Three commands. That's it."
          description="From zero to a fully operational cloud platform in minutes. No DevOps degree required."
        />

        <div className="grid gap-8 md:grid-cols-3">
          {/* Step 1 */}
          <div className="relative">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-accent-500/10 border border-accent-500/20 text-sm font-bold text-accent-400">
                1
              </div>
              <h3 className="text-lg font-semibold text-white">Install</h3>
            </div>
            <div className="rounded-lg border border-border bg-surface-50 p-4 font-mono text-sm">
              <span className="text-accent-400">$</span>
              <span className="ml-2 text-neutral-200">zen install --provider hetzner</span>
            </div>
            <p className="mt-3 text-sm text-neutral-500">
              One command provisions your management cluster on Hetzner Cloud. Zenith operator, gateway, monitoring -- all installed automatically.
            </p>
            {/* Connector line (desktop) */}
            <div className="absolute right-0 top-5 hidden h-px w-8 translate-x-full bg-gradient-to-r from-accent-500/40 to-transparent md:block" />
          </div>

          {/* Step 2 */}
          <div className="relative">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-accent-500/10 border border-accent-500/20 text-sm font-bold text-accent-400">
                2
              </div>
              <h3 className="text-lg font-semibold text-white">Deploy</h3>
            </div>
            <div className="rounded-lg border border-border bg-surface-50 p-4 font-mono text-sm">
              <span className="text-accent-400">$</span>
              <span className="ml-2 text-neutral-200">cd my-app && zen deploy</span>
            </div>
            <p className="mt-3 text-sm text-neutral-500">
              Push your app to Zenith. It detects your framework, builds it, sets up TLS, and gives you a URL. Zero config needed.
            </p>
            <div className="absolute right-0 top-5 hidden h-px w-8 translate-x-full bg-gradient-to-r from-accent-500/40 to-transparent md:block" />
          </div>

          {/* Step 3 */}
          <div>
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-accent-500/10 border border-accent-500/20 text-sm font-bold text-accent-400">
                3
              </div>
              <h3 className="text-lg font-semibold text-white">Scale</h3>
            </div>
            <div className="rounded-lg border border-border bg-surface-50 p-4 font-mono text-sm">
              <span className="text-accent-400">$</span>
              <span className="ml-2 text-neutral-200">zen scale my-app --replicas 5</span>
            </div>
            <p className="mt-3 text-sm text-neutral-500">
              Everything auto-scales. Or fine-tune it yourself. CAPI manages nodes, the operator handles the rest. Sit back and relax.
            </p>
          </div>
        </div>
      </Section>

      {/* ===== PRICING SECTION ===== */}
      <Section id="pricing" className="border-t border-border">
        <SectionHeader
          label="Pricing"
          title="Free forever. Really."
          description="Zenith is 100% open source with no feature gating. Pay only for Hetzner infrastructure."
        />

        <div className="grid gap-6 md:grid-cols-3">
          <PricingCard
            name="Free"
            price="$0"
            period="forever"
            description="Everything included. Self-hosted on your Hetzner account."
            features={[
              "All platform features",
              "Unlimited apps and databases",
              "Built-in auth, gateway, monitoring",
              "CLI and web dashboard",
              "Community support",
              "MIT licensed",
            ]}
            cta="Get Started"
            ctaHref="#get-started"
            featured
          />
          <PricingCard
            name="Pro Support"
            price="$49"
            period="/month"
            description="Priority support for teams running Zenith in production."
            features={[
              "Everything in Free",
              "Priority email support",
              "48-hour response SLA",
              "Assisted upgrades",
              "Architecture review",
              "Private Discord channel",
            ]}
            cta="Contact Sales"
            ctaHref="mailto:support@freezenith.com"
          />
          <PricingCard
            name="Enterprise"
            price="Custom"
            description="Dedicated support for large-scale deployments."
            features={[
              "Everything in Pro Support",
              "Dedicated support engineer",
              "Custom SLAs",
              "White-label option",
              "On-call incident response",
              "Custom feature development",
            ]}
            cta="Talk to Us"
            ctaHref="mailto:enterprise@freezenith.com"
          />
        </div>
      </Section>

      {/* ===== ARCHITECTURE SECTION ===== */}
      <Section id="architecture" className="border-t border-border">
        <SectionHeader
          label="Architecture"
          title="Built on proven technology"
          description="Zenith combines the best open-source tools into a unified platform managed by a Kubernetes operator."
        />

        <div className="mx-auto max-w-3xl">
          {/* Architecture diagram using CSS */}
          <div className="rounded-xl border border-border bg-surface-50 p-6 md:p-8">
            {/* Layer 1: Internet */}
            <div className="text-center">
              <div className="inline-flex items-center gap-2 rounded-lg border border-border bg-surface-200 px-4 py-2">
                <Network className="h-4 w-4 text-neutral-400" />
                <span className="text-sm font-medium text-neutral-300">Internet</span>
              </div>
            </div>

            {/* Connector */}
            <div className="mx-auto my-3 h-6 w-px bg-gradient-to-b from-neutral-600 to-accent-500/40" />

            {/* Layer 2: Load Balancer */}
            <div className="text-center">
              <div className="inline-flex items-center gap-2 rounded-lg border border-accent-500/30 bg-accent-500/5 px-4 py-2">
                <Scaling className="h-4 w-4 text-accent-400" />
                <span className="text-sm font-medium text-accent-300">Hetzner Load Balancer</span>
              </div>
            </div>

            {/* Connector */}
            <div className="mx-auto my-3 h-6 w-px bg-gradient-to-b from-accent-500/40 to-accent-500/20" />

            {/* Layer 3: Kong Gateway */}
            <div className="text-center">
              <div className="inline-flex items-center gap-2 rounded-lg border border-accent-500/20 bg-accent-500/5 px-4 py-2">
                <Shield className="h-4 w-4 text-accent-400" />
                <span className="text-sm font-medium text-accent-300">Kong API Gateway</span>
              </div>
              <p className="mt-1 text-xs text-neutral-500">JWT validation, rate limiting, routing</p>
            </div>

            {/* Connector */}
            <div className="mx-auto my-3 h-6 w-px bg-gradient-to-b from-accent-500/20 to-accent-500/10" />

            {/* Layer 4: Kubernetes */}
            <div className="rounded-lg border border-border bg-surface-100 p-4 md:p-6">
              <div className="mb-4 flex items-center justify-center gap-2">
                <Layers className="h-4 w-4 text-accent-400" />
                <span className="text-sm font-semibold text-white">Kubernetes (k3s / CAPI)</span>
              </div>

              {/* Inner grid: Zenith components */}
              <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
                <ArchBlock icon={Box} label="Zenith Operator" accent />
                <ArchBlock icon={Shield} label="Auth Service" />
                <ArchBlock icon={BarChart3} label="Monitoring" />
                <ArchBlock icon={Database} label="DB Operators" />
              </div>

              {/* Connector */}
              <div className="mx-auto my-3 h-4 w-px bg-border" />

              {/* Your apps */}
              <div className="rounded-lg border border-dashed border-accent-500/30 bg-accent-500/5 p-3 text-center">
                <div className="flex items-center justify-center gap-2">
                  <Rocket className="h-4 w-4 text-accent-400" />
                  <span className="text-sm font-medium text-accent-300">Your Apps & Services</span>
                </div>
                <p className="mt-1 text-xs text-neutral-500">Containers, functions, cron jobs</p>
              </div>
            </div>

            {/* Connector */}
            <div className="mx-auto my-3 h-6 w-px bg-border" />

            {/* Layer 5: Hetzner */}
            <div className="text-center">
              <div className="inline-flex items-center gap-2 rounded-lg border border-border bg-surface-200 px-4 py-2">
                <Server className="h-4 w-4 text-neutral-400" />
                <span className="text-sm font-medium text-neutral-300">Hetzner Cloud</span>
              </div>
              <p className="mt-1 text-xs text-neutral-500">Servers, volumes, networking, DNS, object storage</p>
            </div>
          </div>
        </div>
      </Section>

      {/* ===== OPEN SOURCE SECTION ===== */}
      <Section className="border-t border-border">
        <div className="mx-auto max-w-3xl text-center">
          <div className="mb-6">
            <div className="inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-accent-500/10 border border-accent-500/20">
              <Github className="h-8 w-8 text-accent-400" />
            </div>
          </div>

          <h2 className="text-3xl font-bold text-white md:text-4xl">
            100% open source. MIT licensed.
          </h2>
          <p className="mx-auto mt-4 max-w-lg text-neutral-400">
            No vendor lock-in. No hidden features. Fork it, modify it, self-host it.
            The entire platform is yours to own.
          </p>

          <div className="mt-8 flex flex-col items-center justify-center gap-4 sm:flex-row sm:gap-6">
            <Link
              href="https://github.com/DoTech/zenith"
              className="group inline-flex items-center gap-2 rounded-lg bg-surface-200 border border-border px-5 py-2.5 text-sm font-medium text-white transition-all hover:border-border-hover hover:bg-surface-300"
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
          </div>

          {/* Tech stack badges */}
          <div className="mt-10 flex flex-wrap items-center justify-center gap-2">
            {["Go", "Kubernetes", "Next.js", "TypeScript", "Helm", "Kong", "Grafana", "PostgreSQL"].map(
              (tech) => (
                <span
                  key={tech}
                  className="rounded-full border border-border bg-surface-100 px-3 py-1 text-xs text-neutral-500"
                >
                  {tech}
                </span>
              )
            )}
          </div>
        </div>
      </Section>

      {/* ===== GET STARTED CTA ===== */}
      <Section id="get-started" className="border-t border-border">
        <div className="relative overflow-hidden rounded-2xl border border-accent-500/20 bg-gradient-to-br from-accent-950/50 via-surface-50 to-surface-50 p-8 md:p-12 text-center">
          {/* Background glow */}
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[400px] h-[200px] bg-accent-500/10 rounded-full blur-3xl" />

          <div className="relative">
            <h2 className="text-3xl font-bold text-white md:text-4xl">
              Ready to deploy?
            </h2>
            <p className="mx-auto mt-4 max-w-lg text-neutral-400">
              Get your entire platform running in under 10 minutes. No credit card, no sign-up.
              Just you and your Hetzner account.
            </p>

            {/* Install command */}
            <div className="mx-auto mt-8 max-w-lg">
              <div className="flex items-center rounded-lg border border-border bg-surface-100 px-4 py-3 font-mono text-sm">
                <span className="text-accent-400">$</span>
                <span className="ml-2 flex-1 text-left text-neutral-200">
                  zen install --provider hetzner --token hc_xxx
                </span>
                <button
                  className="ml-2 rounded p-1 text-neutral-500 transition-colors hover:text-white"
                  aria-label="Copy to clipboard"
                >
                  <TerminalIcon className="h-4 w-4" />
                </button>
              </div>
            </div>

            <div className="mt-6 flex flex-col items-center justify-center gap-3 sm:flex-row sm:gap-4">
              <Link
                href="https://github.com/DoTech/zenith"
                className="group inline-flex items-center gap-2 rounded-lg bg-accent-500 px-6 py-3 text-sm font-semibold text-white transition-all hover:bg-accent-600 hover:shadow-lg hover:shadow-accent-500/25"
                target="_blank"
                rel="noopener noreferrer"
              >
                Read the Docs
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </Link>
              <Link
                href="/docs"
                className="text-sm text-neutral-400 transition-colors hover:text-white"
              >
                Or browse the documentation
              </Link>
            </div>
          </div>
        </div>
      </Section>
    </div>
  );
}

/* ===== Architecture diagram helper ===== */

function ArchBlock({
  icon: Icon,
  label,
  accent = false,
}: {
  icon: typeof Box;
  label: string;
  accent?: boolean;
}) {
  return (
    <div
      className={`flex flex-col items-center gap-1.5 rounded-lg border p-3 text-center ${
        accent
          ? "border-accent-500/30 bg-accent-500/5"
          : "border-border bg-surface-200"
      }`}
    >
      <Icon className={`h-4 w-4 ${accent ? "text-accent-400" : "text-neutral-400"}`} />
      <span className={`text-xs font-medium ${accent ? "text-accent-300" : "text-neutral-400"}`}>
        {label}
      </span>
    </div>
  );
}
