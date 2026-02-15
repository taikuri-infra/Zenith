"use client";

import { Section, SectionHeader } from "@/components/section";
import { PricingCard } from "@/components/pricing-card";
import { CostCalculator } from "@/components/cost-calculator";
import Link from "next/link";
import { motion, useInView } from "framer-motion";
import { useRef, useState } from "react";
import { ArrowRight, Check, ChevronDown, ChevronUp, HelpCircle } from "lucide-react";

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
    a: "Yes. Zenith is 100% open source under the MIT license. All features are included with no restrictions. You only pay for Hetzner Cloud infrastructure (starting at ~4.50 EUR/month for a small server).",
  },
  {
    q: "How much does Hetzner infrastructure cost?",
    a: "A minimal Zenith setup runs on a single CX22 (2 vCPU, 4 GB RAM) at ~4.50 EUR/month. Production setups with multiple worker nodes typically cost 20-50 EUR/month. Use the cost calculator above to estimate your specific needs.",
  },
  {
    q: "What does Pro Support include?",
    a: "Pro Support gives you priority email support with a 48-hour response SLA, assisted upgrades, architecture reviews, and a private Discord channel for your team.",
  },
  {
    q: "Can I switch plans later?",
    a: "Absolutely. Since the software is identical across all plans, you can start free and add Pro Support whenever you need it. Downgrading is just as easy -- you never lose access to any features.",
  },
  {
    q: "How does Zenith compare to AWS/GCP cost-wise?",
    a: "For a typical setup with 10 apps, 5 databases, and 100GB storage, Zenith on Hetzner costs approximately 20 EUR/month compared to $500+ on AWS. That is up to 96% savings, with the same capabilities.",
  },
  {
    q: "Do you offer a managed version?",
    a: "Not yet. Zenith is self-hosted only for now. A managed offering is on our roadmap -- join the waitlist to be the first to know.",
  },
];

function FAQItem({ q, a, index }: { q: string; a: string; index: number }) {
  const [open, setOpen] = useState(false);
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true, margin: "-50px" });

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 15 }}
      animate={isInView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.4, delay: index * 0.05 }}
      className="rounded-xl border border-border bg-surface-50/50 transition-all duration-300 hover:border-border-hover overflow-hidden"
    >
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-start gap-3 p-5 text-left"
      >
        <HelpCircle className="mt-0.5 h-4 w-4 shrink-0 text-accent-400" />
        <span className="flex-1 text-sm font-semibold text-white">{q}</span>
        <div className="shrink-0 text-neutral-500 transition-transform">
          {open ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
        </div>
      </button>
      <div
        className={`overflow-hidden transition-all duration-300 ${
          open ? "max-h-96 pb-5" : "max-h-0"
        }`}
      >
        <div className="px-5 pl-12">
          <p className="text-sm leading-relaxed text-neutral-400">{a}</p>
        </div>
      </div>
    </motion.div>
  );
}

export default function PricingPage() {
  const tableRef = useRef(null);
  const tableInView = useInView(tableRef, { once: true, margin: "-100px" });

  return (
    <div className="pt-24">
      {/* Pricing cards */}
      <Section>
        <div className="text-center mb-16">
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="mb-4"
          >
            <span className="inline-flex items-center gap-2 rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 text-xs font-medium uppercase tracking-wide text-accent-400">
              Pricing
            </span>
          </motion.div>
          <motion.h1
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
            className="text-3xl font-bold tracking-tight text-white md:text-4xl lg:text-5xl"
          >
            Simple, transparent pricing
          </motion.h1>
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
            className="mx-auto mt-4 max-w-xl text-neutral-400 leading-relaxed"
          >
            The platform is free forever. Support plans are optional for teams that need guaranteed response times.
          </motion.p>
        </div>

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
            index={0}
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
            index={1}
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
            index={2}
          />
        </div>
      </Section>

      {/* Cost Calculator */}
      <Section className="border-t border-border/50">
        <SectionHeader
          label="Calculator"
          title="Estimate your infrastructure cost"
          description="See how much you will pay for Hetzner Cloud infrastructure based on your workload."
        />
        <CostCalculator />
      </Section>

      {/* Feature comparison table */}
      <Section className="border-t border-border/50">
        <SectionHeader
          title="Feature comparison"
          description="Every feature is available on every plan. Support plans add response time guarantees."
        />

        <motion.div
          ref={tableRef}
          initial={{ opacity: 0, y: 20 }}
          animate={tableInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.6 }}
          className="overflow-x-auto rounded-2xl border border-border"
        >
          <table className="w-full min-w-[600px] text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-50/50">
                <th className="p-4 text-left font-medium text-neutral-400">Feature</th>
                <th className="p-4 text-center">
                  <span className="font-semibold text-accent-400">Free</span>
                </th>
                <th className="p-4 text-center">
                  <span className="font-semibold text-white">Pro Support</span>
                </th>
                <th className="p-4 text-center">
                  <span className="font-semibold text-white">Enterprise</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {comparisonFeatures.map((row, i) => (
                <tr key={row.feature} className={`border-b border-border/40 ${i % 2 === 0 ? "bg-surface-50/20" : ""}`}>
                  <td className="p-4 text-neutral-300">{row.feature}</td>
                  <td className="p-4 text-center">
                    {row.free ? (
                      <div className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-accent-500/10">
                        <Check className="h-3 w-3 text-accent-400" />
                      </div>
                    ) : (
                      <span className="text-neutral-700">--</span>
                    )}
                  </td>
                  <td className="p-4 text-center">
                    {row.pro ? (
                      <div className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-accent-500/10">
                        <Check className="h-3 w-3 text-accent-400" />
                      </div>
                    ) : (
                      <span className="text-neutral-700">--</span>
                    )}
                  </td>
                  <td className="p-4 text-center">
                    {row.enterprise ? (
                      <div className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-accent-500/10">
                        <Check className="h-3 w-3 text-accent-400" />
                      </div>
                    ) : (
                      <span className="text-neutral-700">--</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </motion.div>
      </Section>

      {/* FAQ */}
      <Section className="border-t border-border/50">
        <SectionHeader title="Frequently asked questions" />

        <div className="mx-auto max-w-2xl space-y-3">
          {faqs.map((faq, i) => (
            <FAQItem key={faq.q} {...faq} index={i} />
          ))}
        </div>
      </Section>

      {/* Bottom CTA */}
      <Section className="border-t border-border/50">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="text-center"
        >
          <h2 className="text-2xl font-bold text-white md:text-3xl lg:text-4xl">
            Ready to get started?
          </h2>
          <p className="mt-4 text-neutral-400 leading-relaxed">
            Install Zenith in under 10 minutes. No credit card required.
          </p>
          <div className="mt-8">
            <Link
              href="/#get-started"
              className="group inline-flex items-center gap-2 rounded-xl bg-accent-500 px-7 py-3.5 text-sm font-semibold text-white transition-all duration-300 hover:bg-accent-600 hover:shadow-xl hover:shadow-accent-500/25"
            >
              Get Started
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Link>
          </div>
        </motion.div>
      </Section>
    </div>
  );
}
