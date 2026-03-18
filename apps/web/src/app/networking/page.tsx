"use client";

import { Shell } from "@/components/shell";
import { Modal } from "@/components/modal";
import { useEffect, useState, useCallback } from "react";
import { getApi } from "@/lib/get-api";

import type { DeployApp } from "@/lib/api";

interface Domain {
  domain: string;
  target: string;
  ssl: string;
  status: string;
}

export default function NetworkingPage() {
  const api = getApi();
  const [domains, setDomains] = useState<Domain[]>([]);
  const [loading, setLoading] = useState(true);

  const [showAddDomain, setShowAddDomain] = useState(false);
  const [domainName, setDomainName] = useState("");
  const [domainTarget, setDomainTarget] = useState("");

  const loadDomains = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await api.appsDeploy.list();
      const apps: DeployApp[] = resp.items;
      // Build domains from deployed apps that have a subdomain
      const appDomains: Domain[] = apps
        .filter((a) => a.subdomain && a.status !== "deleted")
        .map((a) => ({
          domain: `${a.subdomain}.apps.stage.freezenith.com`,
          target: `${a.name}:${a.port || 8080}`,
          ssl: "Active",
          status: a.status === "running" ? "active" : a.status === "deploying" ? "pending" : a.status,
        }));
      setDomains(appDomains);
    } catch {
      setDomains([]);
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    loadDomains();
  }, [loadDomains]);

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
            Domains and routing for your deployed apps
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
          {loading ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center text-sm text-neutral-500">
              Loading...
            </div>
          ) : domains.length === 0 ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
              <p className="text-sm text-neutral-400">No domains configured yet.</p>
              <p className="mt-1 text-xs text-neutral-600">Deploy an app to get an automatic subdomain, or add a custom domain.</p>
            </div>
          ) : (
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
          )}
        </section>

        {/* Load Balancers */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Load Balancers</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
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
                  {domains.length > 0 ? "Load balancer active" : "No traffic configured"}
                </p>
                <p className="text-xs text-neutral-500">
                  {domains.length > 0
                    ? `Distributing traffic across ${domains.length} configured domains.`
                    : "Deploy apps to enable automatic load balancing."}
                </p>
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
    </Shell>
  );
}
