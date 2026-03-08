"use client";

import { useState } from "react";
import { Shell } from "@/components/shell";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { AddOn } from "@/lib/api";
import {
  Headphones,
  Cpu,
  HardDrive,
  Shield,
  Network,
  Check,
  Star,
  Lock,
} from "lucide-react";

const categoryMeta: Record<
  string,
  { label: string; icon: React.ComponentType<{ className?: string }>; color: string }
> = {
  support: { label: "Support", icon: Headphones, color: "text-amber-400" },
  compute: { label: "Compute", icon: Cpu, color: "text-blue-400" },
  storage: { label: "Storage", icon: HardDrive, color: "text-green-400" },
  security: { label: "Security", icon: Shield, color: "text-red-400" },
  network: { label: "Network", icon: Network, color: "text-purple-400" },
};

const categories = ["all", "support", "compute", "storage", "security", "network"];

function formatPrice(cents: number): string {
  return `\u20AC${(cents / 100).toFixed(cents % 100 === 0 ? 0 : 2)}`;
}

export default function MarketplacePage() {
  const [filter, setFilter] = useState("all");
  const [selectedAddon, setSelectedAddon] = useState<AddOn | null>(null);

  const api = getApi();
  const { data: addons, loading } = useApi(() => api.addons.list(), []);

  const filtered = addons?.filter(
    (a) => filter === "all" || a.category === filter
  );

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Marketplace</h1>
          <p className="text-sm text-neutral-500">
            Enhance your platform with add-ons and premium services
          </p>
        </div>

        {/* Category filters */}
        <div className="flex flex-wrap gap-2">
          {categories.map((cat) => {
            const meta = categoryMeta[cat];
            return (
              <button
                key={cat}
                onClick={() => setFilter(cat)}
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                  filter === cat
                    ? "bg-white/10 text-white"
                    : "text-neutral-500 hover:text-neutral-300 hover:bg-white/5"
                }`}
              >
                {cat === "all" ? "All" : meta?.label ?? cat}
              </button>
            );
          })}
        </div>

        {/* Loading state */}
        {loading && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <div
                key={i}
                className="h-52 animate-pulse rounded-xl border border-border bg-surface-100"
              />
            ))}
          </div>
        )}

        {/* Add-on grid */}
        {!loading && filtered && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((addon) => {
              const meta = categoryMeta[addon.category];
              const Icon = meta?.icon ?? Shield;
              return (
                <button
                  key={addon.id}
                  onClick={() => setSelectedAddon(addon)}
                  className={`group relative rounded-xl border border-border bg-surface-100 p-5 text-left transition-all hover:border-white/20 hover:bg-surface-200 ${
                    !addon.available ? "opacity-60" : ""
                  }`}
                >
                  {addon.popular && (
                    <span className="absolute -top-2 right-3 flex items-center gap-1 rounded-full bg-amber-500/20 px-2 py-0.5 text-[10px] font-semibold text-amber-400">
                      <Star className="h-2.5 w-2.5" /> Popular
                    </span>
                  )}
                  <div className="flex items-start gap-3">
                    <div
                      className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-surface-300 ${
                        meta?.color ?? "text-neutral-400"
                      }`}
                    >
                      <Icon className="h-5 w-5" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <h3 className="text-sm font-medium text-white">
                        {addon.name}
                      </h3>
                      <span
                        className={`text-[10px] font-medium ${
                          meta?.color ?? "text-neutral-400"
                        }`}
                      >
                        {meta?.label ?? addon.category}
                      </span>
                    </div>
                    <div className="text-right">
                      <span className="text-sm font-semibold text-white">
                        {formatPrice(addon.price_cents)}
                      </span>
                      <span className="block text-[10px] text-neutral-500">
                        /month
                      </span>
                    </div>
                  </div>
                  <p className="mt-3 text-xs leading-relaxed text-neutral-400">
                    {addon.description}
                  </p>
                  {!addon.available && (
                    <div className="mt-3 flex items-center gap-1 text-[10px] text-neutral-500">
                      <Lock className="h-3 w-3" />
                      Requires {addon.min_tier} plan or higher
                    </div>
                  )}
                </button>
              );
            })}
          </div>
        )}

        {/* Detail modal */}
        {selectedAddon && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div className="w-full max-w-lg rounded-2xl border border-border bg-surface-100 p-6 shadow-2xl">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  {(() => {
                    const meta = categoryMeta[selectedAddon.category];
                    const Icon = meta?.icon ?? Shield;
                    return (
                      <div
                        className={`flex h-12 w-12 items-center justify-center rounded-xl bg-surface-300 ${
                          meta?.color ?? "text-neutral-400"
                        }`}
                      >
                        <Icon className="h-6 w-6" />
                      </div>
                    );
                  })()}
                  <div>
                    <h2 className="text-base font-semibold text-white">
                      {selectedAddon.name}
                    </h2>
                    <span className="text-xs text-neutral-500">
                      {formatPrice(selectedAddon.price_cents)}/month
                    </span>
                  </div>
                </div>
                <button
                  onClick={() => setSelectedAddon(null)}
                  className="text-neutral-500 hover:text-white transition-colors text-lg"
                >
                  &times;
                </button>
              </div>

              <p className="mt-4 text-sm text-neutral-400">
                {selectedAddon.description}
              </p>

              <div className="mt-5">
                <h3 className="text-xs font-semibold uppercase tracking-wider text-neutral-500 mb-3">
                  Features
                </h3>
                <ul className="space-y-2">
                  {selectedAddon.features.map((f) => (
                    <li
                      key={f}
                      className="flex items-center gap-2 text-sm text-neutral-300"
                    >
                      <Check className="h-3.5 w-3.5 shrink-0 text-green-400" />
                      {f}
                    </li>
                  ))}
                </ul>
              </div>

              {!selectedAddon.available && (
                <div className="mt-5 rounded-lg border border-amber-500/20 bg-amber-500/5 px-4 py-3 text-xs text-amber-400">
                  <Lock className="mr-1.5 inline h-3 w-3" />
                  This add-on requires the{" "}
                  <span className="font-semibold capitalize">
                    {selectedAddon.min_tier}
                  </span>{" "}
                  plan or higher. Upgrade your plan to enable this add-on.
                </div>
              )}

              <div className="mt-6 flex gap-3">
                <button
                  onClick={() => setSelectedAddon(null)}
                  className="flex-1 rounded-lg border border-border bg-surface-200 py-2.5 text-xs font-medium text-neutral-300 hover:bg-surface-300 transition-colors"
                >
                  Close
                </button>
                <button
                  disabled={!selectedAddon.available}
                  className={`flex-1 rounded-lg py-2.5 text-xs font-medium transition-colors ${
                    selectedAddon.available
                      ? "bg-white text-black hover:bg-neutral-200"
                      : "cursor-not-allowed bg-surface-300 text-neutral-500"
                  }`}
                >
                  {selectedAddon.available ? "Subscribe" : "Upgrade Plan"}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}
