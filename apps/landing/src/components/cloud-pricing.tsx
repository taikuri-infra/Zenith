"use client";

import { PricingCard } from "@/components/pricing-card";
import { registerUrl } from "@/lib/urls";

export function CloudPricing() {
  return (
    <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-5">
      <PricingCard
        name="Free"
        price="€0"
        period="forever"
        description="Perfect for trying things out. No credit card required."
        features={[
          "1 app",
          "1 database (500MB)",
          "1GB storage",
          "Sleep after 15 min idle",
          "Community support",
        ]}
        cta="Start Free"
        ctaHref={registerUrl}
        index={0}
      />
      <PricingCard
        name="Pro"
        price="€29"
        period="/mo"
        description="For solo devs and early-stage startups."
        features={[
          "5 apps",
          "3 databases (5GB each)",
          "10GB storage",
          "Custom domains",
          "Always on — no sleep",
          "Automated backups",
        ]}
        cta="Upgrade to Pro"
        ctaHref={registerUrl}
        index={1}
      />
      <PricingCard
        name="Team"
        price="€99"
        period="/seat/mo"
        description="For growing teams with collaboration needs."
        features={[
          "20 apps",
          "10 databases (20GB each)",
          "100GB storage",
          "RBAC + SSO",
          "Up to 10 members",
          "Priority support",
        ]}
        cta="Start Team Trial"
        ctaHref={registerUrl}
        index={2}
      />
      <PricingCard
        name="Business"
        price="€149"
        period="/seat/mo"
        description="For funded startups needing dedicated infra."
        features={[
          "Unlimited apps",
          "Dedicated infrastructure",
          "Audit log + compliance",
          "50 team members",
          "IP whitelisting",
          "White-label branding",
        ]}
        cta="Start Business Trial"
        ctaHref={registerUrl}
        featured
        index={3}
      />
      <PricingCard
        name="Enterprise"
        price="Custom"
        description="Full isolation with compliance and SLAs."
        features={[
          "Unlimited everything",
          "Dedicated namespace",
          "Custom SLAs",
          "SOC 2, GDPR, ISO 27001",
          "Dedicated support engineer",
          "Custom integrations",
        ]}
        cta="Talk to Us"
        ctaHref="mailto:enterprise@freezenith.com"
        index={4}
      />
    </div>
  );
}
