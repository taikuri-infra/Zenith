"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import { CloudPricing } from "@/components/cloud-pricing";
import { PricingComparison } from "@/components/pricing-comparison";
import { CostCalculator } from "@/components/cost-calculator";

interface PricingTabsProps {
  defaultTab?: "cloud" | "self-hosted";
  showCalculator?: boolean;
  showComparison?: boolean;
}

export function PricingTabs({
  defaultTab = "cloud",
  showCalculator = false,
  showComparison = false,
}: PricingTabsProps) {
  const [activeTab, setActiveTab] = useState<"cloud" | "self-hosted">(defaultTab);

  return (
    <div>
      {/* Tab switcher */}
      <div className="mb-12 flex justify-center">
        <div className="inline-flex rounded-full border border-border bg-surface-50/80 p-1">
          <button
            onClick={() => setActiveTab("cloud")}
            className={cn(
              "rounded-full px-5 py-2 text-sm font-medium transition-all duration-200",
              activeTab === "cloud"
                ? "bg-accent-500 text-white shadow-lg shadow-accent-500/25"
                : "text-neutral-400 hover:text-white"
            )}
          >
            Cloud
          </button>
          <button
            onClick={() => setActiveTab("self-hosted")}
            className={cn(
              "rounded-full px-5 py-2 text-sm font-medium transition-all duration-200",
              activeTab === "self-hosted"
                ? "bg-accent-500 text-white shadow-lg shadow-accent-500/25"
                : "text-neutral-400 hover:text-white"
            )}
          >
            Self-Hosted
          </button>
        </div>
      </div>

      {/* Tab content */}
      {activeTab === "cloud" ? (
        <CloudPricing />
      ) : (
        <div className="space-y-16">
          <PricingComparison />
          {showCalculator && <CostCalculator />}
          {showComparison && (
            <p className="text-center text-sm text-neutral-500">
              Zenith is 100% free and open source. You only pay for Hetzner infrastructure.
            </p>
          )}
        </div>
      )}
    </div>
  );
}
