import { Section, SectionHeader } from "@/components/section";
import { PricingCard } from "@/components/pricing-card";
import Link from "next/link";
import { ArrowRight, Check, HelpCircle } from "lucide-react";

const comparisonFeatures = [
  { feature: "Apps (unlimited)", free: true, pro: true, enterprise: true },
  { feature: "Databases (PostgreSQL, MySQL, MongoDB, Redis)", free: true, pro: true, enterprise: true },
  { feature: "Built-in Auth (OAuth, SAML, MFA)", free: true, pro: true, enterprise: true },
  { feature: "API Gateway (Kong)", free: true, pro: true, enterprise: true },
  { feature: "Monitoring (Grafana, Prometheus, Loki)", free: true, pro: true, enterprise: true },
  { feature: "S3-compatible Storage", free: true, pro: true, enterprise: true },
  { feature: "CLI + Web Dashboard", free: true, pro: true, enterprise: true },
  { feature: "Custom Domains + TLS", free: true, pro: true, enterprise: true },
  { feature: "Auto-scaling", free: true, pro: true, enterprise: true },
  { feature: "GitOps (zen export/apply/diff)", free: true, pro: true, enterprise: true },
  { feature: "Community Support", free: true, pro: true, enterprise: true },
  { feature: "Priority Email Support", free: false, pro: true, enterprise: true },
  { feature: "Response SLA (48h)", free: false, pro: true, enterprise: true },
  { feature: "Assisted Upgrades", free: false, pro: true, enterprise: true },
  { feature: "Architecture Review", free: false, pro: true, enterprise: true },
  { feature: "Private Discord Channel", free: false, pro: true, enterprise: true },
  { feature: "Dedicated Support Engineer", free: false, pro: false, enterprise: true },
  { feature: "Custom SLAs", free: false, pro: false, enterprise: true },
  { feature: "White-label Option", free: false, pro: false, enterprise: true },
  { feature: "On-call Incident Response", free: false, pro: false, enterprise: true },
  { feature: "Custom Feature Development", free: false, pro: false, enterprise: true },
];

const faqs = [
  {
    q: "Is Zenith really free?",
    a: "Yes. Zenith is 100% open source under the MIT license. All features are included. You only pay for Hetzner Cloud infrastructure (starting at ~5 EUR/month for a small server).",
  },
  {
    q: "What does Pro Support include?",
    a: "Pro Support gives you priority email support with a 48-hour response SLA, assisted upgrades, architecture reviews, and a private Discord channel for your team.",
  },
  {
    q: "Can I switch plans later?",
    a: "Absolutely. Since the software is the same, you can start free and add Pro Support when you need it. Downgrading is just as easy.",
  },
  {
    q: "How much does Hetzner infrastructure cost?",
    a: "A minimal Zenith setup runs on a single CX22 (2 vCPU, 4 GB RAM) at ~4.50 EUR/month. Production setups with multiple worker nodes typically cost 20-50 EUR/month.",
  },
  {
    q: "Do you offer a managed version?",
    a: "Not yet. Zenith is self-hosted only for now. A managed offering is on our roadmap. Join the waitlist to be notified.",
  },
];

export default function PricingPage() {
  return (
    <div className="pt-20">
      {/* Pricing cards */}
      <Section>
        <SectionHeader
          label="Pricing"
          title="Simple, transparent pricing"
          description="The platform is free forever. Support plans are optional for teams that need guaranteed response times."
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
            ctaHref="/#get-started"
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

      {/* Feature comparison table */}
      <Section className="border-t border-border">
        <SectionHeader
          title="Feature comparison"
          description="Every feature is available on every plan. Support plans add response time guarantees."
        />

        <div className="overflow-x-auto">
          <table className="w-full min-w-[600px] text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="pb-4 text-left font-medium text-neutral-400">Feature</th>
                <th className="pb-4 text-center font-medium text-white">Free</th>
                <th className="pb-4 text-center font-medium text-accent-400">Pro Support</th>
                <th className="pb-4 text-center font-medium text-white">Enterprise</th>
              </tr>
            </thead>
            <tbody>
              {comparisonFeatures.map((row) => (
                <tr key={row.feature} className="border-b border-border/50">
                  <td className="py-3 text-neutral-300">{row.feature}</td>
                  <td className="py-3 text-center">
                    {row.free ? (
                      <Check className="mx-auto h-4 w-4 text-accent-400" />
                    ) : (
                      <span className="text-neutral-600">--</span>
                    )}
                  </td>
                  <td className="py-3 text-center">
                    {row.pro ? (
                      <Check className="mx-auto h-4 w-4 text-accent-400" />
                    ) : (
                      <span className="text-neutral-600">--</span>
                    )}
                  </td>
                  <td className="py-3 text-center">
                    {row.enterprise ? (
                      <Check className="mx-auto h-4 w-4 text-accent-400" />
                    ) : (
                      <span className="text-neutral-600">--</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Section>

      {/* FAQ */}
      <Section className="border-t border-border">
        <SectionHeader
          title="Frequently asked questions"
        />

        <div className="mx-auto max-w-2xl space-y-6">
          {faqs.map((faq) => (
            <div key={faq.q} className="rounded-lg border border-border bg-surface-50 p-5">
              <div className="flex items-start gap-3">
                <HelpCircle className="mt-0.5 h-4 w-4 shrink-0 text-accent-400" />
                <div>
                  <h3 className="text-sm font-semibold text-white">{faq.q}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-neutral-400">{faq.a}</p>
                </div>
              </div>
            </div>
          ))}
        </div>
      </Section>

      {/* Bottom CTA */}
      <Section className="border-t border-border">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-white md:text-3xl">Ready to get started?</h2>
          <p className="mt-3 text-neutral-400">
            Install Zenith in under 10 minutes. No credit card required.
          </p>
          <div className="mt-6">
            <Link
              href="/#get-started"
              className="group inline-flex items-center gap-2 rounded-lg bg-accent-500 px-6 py-3 text-sm font-semibold text-white transition-all hover:bg-accent-600 hover:shadow-lg hover:shadow-accent-500/25"
            >
              Get Started
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Link>
          </div>
        </div>
      </Section>
    </div>
  );
}
