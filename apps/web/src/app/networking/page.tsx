"use client";

import { Shell } from "@/components/shell";
import { Modal } from "@/components/modal";
import { useState } from "react";

interface Domain {
  domain: string;
  target: string;
  ssl: string;
  status: string;
}

interface FirewallRule {
  port: string;
  protocol: string;
  source: string;
  action: string;
}

const initialDomains: Domain[] = [
  { domain: "app.startup.com", target: "frontend:3000", ssl: "Active", status: "active" },
  { domain: "api.startup.com", target: "api-gateway:8080", ssl: "Active", status: "active" },
  { domain: "admin.startup.com", target: "admin-panel:3000", ssl: "Pending", status: "pending" },
];

const initialFirewallRules: FirewallRule[] = [
  { port: "80", protocol: "TCP", source: "0.0.0.0/0", action: "Allow" },
  { port: "443", protocol: "TCP", source: "0.0.0.0/0", action: "Allow" },
  { port: "22", protocol: "TCP", source: "10.0.0.0/8", action: "Allow" },
  { port: "*", protocol: "*", source: "0.0.0.0/0", action: "Deny" },
];

export default function NetworkingPage() {
  const [domains, setDomains] = useState<Domain[]>(initialDomains);
  const [firewallRules] = useState<FirewallRule[]>(initialFirewallRules);

  const [showAddDomain, setShowAddDomain] = useState(false);
  const [domainName, setDomainName] = useState("");
  const [domainTarget, setDomainTarget] = useState("");

  const [showFirewall, setShowFirewall] = useState(false);

  const handleAddDomain = () => {
    if (!domainName.trim()) return;
    const newDomain: Domain = {
      domain: domainName.trim(),
      target: domainTarget.trim() || "service:3000",
      ssl: "Pending",
      status: "pending",
    };
    setDomains((prev) => [...prev, newDomain]);
    setShowAddDomain(false);
    setDomainName("");
    setDomainTarget("");
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Networking</h1>
          <p className="text-sm text-neutral-500">
            Domains, firewalls, and load balancers
          </p>
        </div>

        {/* Domains */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Domains</h2>
            <button
              onClick={() => setShowAddDomain(true)}
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
            >
              + Add Domain
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Domain</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Target Service</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">SSL</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {domains.map((d) => (
                  <tr key={d.domain} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{d.domain}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{d.target}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center gap-1.5 text-xs ${d.ssl === "Active" ? "text-emerald-400" : "text-amber-400"}`}>
                        <span className={`h-1.5 w-1.5 rounded-full ${d.ssl === "Active" ? "bg-emerald-400" : "bg-amber-400"}`} />
                        {d.ssl}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium capitalize ${
                        d.status === "active" ? "bg-emerald-500/10 text-emerald-400" : "bg-amber-500/10 text-amber-400"
                      }`}>
                        <span className={`h-1.5 w-1.5 rounded-full ${d.status === "active" ? "bg-emerald-400" : "bg-amber-400"}`} />
                        {d.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Firewalls */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Firewalls</h2>
            <button
              onClick={() => setShowFirewall(true)}
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
            >
              Configure
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Port</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Protocol</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Source</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Action</th>
                </tr>
              </thead>
              <tbody>
                {firewallRules.map((rule, i) => (
                  <tr key={i} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-mono text-xs text-white">{rule.port}</td>
                    <td className="px-4 py-3 text-xs text-neutral-300">{rule.protocol}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{rule.source}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                        rule.action === "Allow" ? "bg-emerald-500/10 text-emerald-400" : "bg-red-500/10 text-red-400"
                      }`}>
                        {rule.action}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Load Balancers */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Load Balancers</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent-500/10">
                  <svg
                    className="h-4.5 w-4.5 text-accent-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                    />
                  </svg>
                </div>
                <div>
                  <p className="text-sm font-medium text-white">
                    Load balancer active
                  </p>
                  <p className="text-xs text-neutral-500">
                    Distributing traffic across {domains.length} configured domains.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>

      {showAddDomain && (
        <Modal title="Add Domain" onClose={() => setShowAddDomain(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleAddDomain();
            }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Domain</label>
              <input
                type="text"
                value={domainName}
                onChange={(e) => setDomainName(e.target.value)}
                placeholder="example.com"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Target Service</label>
              <input
                type="text"
                value={domainTarget}
                onChange={(e) => setDomainTarget(e.target.value)}
                placeholder="service:3000"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => setShowAddDomain(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                Add Domain
              </button>
            </div>
          </form>
        </Modal>
      )}

      {showFirewall && (
        <Modal title="Firewall Configuration" onClose={() => setShowFirewall(false)}>
          <div className="space-y-3">
            <p className="text-xs text-neutral-400">Current firewall rules for this project:</p>
            <div className="overflow-hidden rounded-md border border-border">
              <table className="w-full text-xs">
                <thead>
                  <tr className="bg-surface-200">
                    <th className="px-3 py-2 text-left text-neutral-500">Port</th>
                    <th className="px-3 py-2 text-left text-neutral-500">Source</th>
                    <th className="px-3 py-2 text-left text-neutral-500">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {firewallRules.map((rule, i) => (
                    <tr key={i} className="border-t border-border">
                      <td className="px-3 py-2 font-mono text-neutral-300">{rule.port}/{rule.protocol}</td>
                      <td className="px-3 py-2 font-mono text-neutral-400">{rule.source}</td>
                      <td className="px-3 py-2">
                        <span className={rule.action === "Allow" ? "text-emerald-400" : "text-red-400"}>{rule.action}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button
                onClick={() => setShowFirewall(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Close
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
