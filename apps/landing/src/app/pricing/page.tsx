"use client";

import { Section, SectionHeader } from "@/components/section";
import { PricingTabs } from "@/components/pricing-tabs";
import Link from "next/link";
import { motion, useInView } from "framer-motion";
import { useRef, useState } from "react";
import { ArrowRight, Check, ChevronDown, ChevronUp, HelpCircle } from "lucide-react";
import { registerUrl } from "@/lib/urls";

const cloudFeatures = [
  { feature: "Apps", free: "1", pro: "5", team: "20", business: "Unlimited", enterprise: "Unlimited" },
  { feature: "Databases", free: "1 (500MB)", pro: "3 (5GB)", team: "10 (20GB)", business: "Unlimited", enterprise: "Unlimited" },
  { feature: "Storage", free: "1GB", pro: "10GB", team: "100GB", business: "500GB", enterprise: "Custom" },
  { feature: "Custom Domains", free: false, pro: true, team: true, business: true, enterprise: true },
  { feature: "Always On", free: false, pro: true, team: true, business: true, enterprise: true },
  { feature: "Auto-scaling", free: false, pro: true, team: true, business: true, enterprise: true },
  { feature: "Automated Backups", free: false, pro: true, team: true, business: true, enterprise: true },
  { feature: "Team Members", free: "1", pro: "1", team: "10", business: "50", enterprise: "Unlimited" },
  { feature: "RBAC + SSO", free: false, pro: false, team: true, business: true, enterprise: true },
  { feature: "Audit Log", free: false, pro: false, team: false, business: true, enterprise: true },
  { feature: "Dedicated Infrastructure", free: false, pro: false, team: false, business: true, enterprise: true },
  { feature: "IP Whitelisting", free: false, pro: false, team: false, business: true, enterprise: true },
  { feature: "White-label Branding", free: false, pro: false, team: false, business: true, enterprise: true },
  { feature: "Compliance (SOC 2, GDPR)", free: false, pro: false, team: false, business: true, enterprise: true },
  { feature: "Custom SLAs", free: false, pro: false, team: false, business: false, enterprise: true },
  { feature: "Priority Support", free: false, pro: false, team: true, business: true, enterprise: true },
  { feature: "Dedicated Support Engineer", free: false, pro: false, team: false, business: false, enterprise: true },
];

const faqs = [
  {
    q: "Is there really a free tier?",
    a: "Yes. The free tier gives you 1 app and 1 database with 500MB storage — no credit card required. Your app sleeps after 15 minutes of idle but wakes up automatically on the next request.",
  },
  {
    q: "How does the free tier compare to the self-hosted version?",
    a: "Zenith Cloud's free tier is limited to 1 app and 1 database. Self-hosted Zenith is MIT-licensed with no limits — you can run as many apps and databases as your infrastructure supports.",
  },
  {
    q: "Can I switch between Cloud and Self-Hosted?",
    a: "Yes. Zenith uses the same deployment format for both. You can export your app configuration from Cloud and import it into a self-hosted cluster.",
  },
  {
    q: "What does the self-hosted version cost?",
    a: "Zenith is 100% free and open source under the MIT license. You only pay for Hetzner Cloud infrastructure (starting at ~€4.50/month for a small server). Use the cost calculator on the Self-Hosted tab to estimate.",
  },
  {
    q: "Can I upgrade or downgrade my plan?",
    a: "Absolutely. You can upgrade or downgrade your Cloud plan at any time from your dashboard. Changes take effect immediately, and billing is prorated.",
  },
  {
    q: "How does Zenith Cloud compare to AWS/GCP cost-wise?",
    a: "For a typical setup with 5 apps and 3 databases, Zenith Pro costs €29/month. Teams start at €99/seat and Business at €149/seat. A comparable setup on AWS would cost $200-300/month. Even self-hosting on Hetzner brings similar savings.",
  },
  {
    q: "Do you offer Enterprise on-premise?",
    a: "Yes. Enterprise customers can choose between managed Cloud with dedicated infrastructure or on-premise self-hosted with our support plan. Contact us for details.",
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

function FeatureCell({ value }: { value: boolean | string }) {
  if (typeof value === "string") {
    return <span className="text-sm text-neutral-300">{value}</span>;
  }
  if (value) {
    return (
      <div className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-accent-500/10">
        <Check className="h-3 w-3 text-accent-400" />
      </div>
    );
  }
  return <span className="text-neutral-700">—</span>;
}

export default function PricingPage() {
  const tableRef = useRef(null);
  const tableInView = useInView(tableRef, { once: true, margin: "-100px" });

  return (
    <div className="pt-24">
      {/* Pricing tabs */}
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
            Start free on Zenith Cloud or self-host the open-source platform on your own servers.
          </motion.p>
        </div>

        <PricingTabs defaultTab="cloud" showCalculator showComparison />
      </Section>

      {/* Cloud feature comparison table */}
      <Section className="border-t border-border/50">
        <SectionHeader
          title="Cloud feature comparison"
          description="See exactly what is included in each plan."
        />

        <motion.div
          ref={tableRef}
          initial={{ opacity: 0, y: 20 }}
          animate={tableInView ? { opacity: 1, y: 0 } : {}}
          transition={{ duration: 0.6 }}
          className="overflow-x-auto rounded-2xl border border-border"
        >
          <table className="w-full min-w-[800px] text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-50/50">
                <th className="p-4 text-left font-medium text-neutral-400">Feature</th>
                <th className="p-4 text-center">
                  <span className="font-semibold text-neutral-300">Free</span>
                </th>
                <th className="p-4 text-center">
                  <span className="font-semibold text-accent-400">Pro</span>
                </th>
                <th className="p-4 text-center">
                  <span className="font-semibold text-white">Team</span>
                </th>
                <th className="p-4 text-center">
                  <span className="font-semibold text-amber-400">Business</span>
                </th>
                <th className="p-4 text-center">
                  <span className="font-semibold text-white">Enterprise</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {cloudFeatures.map((row, i) => (
                <tr key={row.feature} className={`border-b border-border/40 ${i % 2 === 0 ? "bg-surface-50/20" : ""}`}>
                  <td className="p-4 text-neutral-300">{row.feature}</td>
                  <td className="p-4 text-center"><FeatureCell value={row.free} /></td>
                  <td className="p-4 text-center"><FeatureCell value={row.pro} /></td>
                  <td className="p-4 text-center"><FeatureCell value={row.team} /></td>
                  <td className="p-4 text-center"><FeatureCell value={row.business} /></td>
                  <td className="p-4 text-center"><FeatureCell value={row.enterprise} /></td>
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
            Deploy your first app in under 5 minutes. No credit card required.
          </p>
          <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row sm:gap-4">
            <Link
              href={registerUrl}
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
        </motion.div>
      </Section>
    </div>
  );
}
