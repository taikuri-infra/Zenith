"use client";
import { appHost } from "@/lib/edition";

import { Shell } from "@/components/shell";
import { DomainStatus } from "@/components/domain-status";
import { useEffect, useState, useCallback } from "react";
import { getApi } from "@/lib/get-api";
import { useToast } from "@/components/toast";

import { customDomains, appsDeploy as realAppsDeploy, type CustomDomain } from "@/lib/api";

interface Domain {
  domain: string;
  target: string;
  ssl: string;
  status: string;
}

export default function NetworkingPage() {
  const api = getApi();
  const { toast } = useToast();
  const [domains, setDomains] = useState<Domain[]>([]);
  const [custom, setCustom] = useState<CustomDomain[]>([]);
  const [appNames, setAppNames] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [checkingId, setCheckingId] = useState<string | null>(null);

  const loadDomains = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await api.appsDeploy.list();
      const apps = resp.items;
      // Build domains from deployed apps that have a subdomain
      const appDomains: Domain[] = apps
        .filter((a) => a.subdomain && a.status !== "deleted")
        .map((a) => ({
          domain: appHost(a),
          target: `${a.name}:${a.port || 8080}`,
          ssl: "Active",
          status: a.status === "running" ? "active" : a.status === "deploying" ? "pending" : a.status,
        }));
      setDomains(appDomains);
      setAppNames(Object.fromEntries(apps.map((a) => [a.id, a.name])));
    } catch {
      setDomains([]);
    }
    try {
      setCustom((await customDomains.list()) ?? []);
    } catch {
      setCustom([]);
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    loadDomains();
  }, [loadDomains]);

  const handleCheck = async (d: CustomDomain) => {
    setCheckingId(d.id);
    try {
      await realAppsDeploy.verifyDomain(d.app_id, d.id);
      await loadDomains();
    } catch {
      toast("error", "Domain check failed");
    } finally {
      setCheckingId(null);
    }
  };

  const empty = domains.length === 0 && custom.length === 0;

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Networking</h1>
          <p className="text-sm text-neutral-500">
            Domains and routing for your deployed apps
          </p>
        </div>

        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Domains</h2>
          </div>
          {loading ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center text-sm text-neutral-500">
              Loading...
            </div>
          ) : empty ? (
            <div className="rounded-lg border border-border bg-surface-100 p-8 text-center">
              <p className="text-sm text-neutral-400">No domains configured yet.</p>
              <p className="mt-1 text-xs text-neutral-600">Add a custom domain from an app&apos;s Domains tab.</p>
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
                  {domains.length > 0 && (
                    <tr className="border-b border-border bg-surface-100/50">
                      <td colSpan={4} className="px-4 py-2 text-xs font-medium text-neutral-400">App subdomains</td>
                    </tr>
                  )}
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
                  {custom.length > 0 && (
                    <tr className="border-b border-border bg-surface-100/50">
                      <td colSpan={4} className="px-4 py-2 text-xs font-medium text-neutral-400">Custom domains</td>
                    </tr>
                  )}
                  {custom.map((d) => (
                    <tr key={d.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                      <td className="px-4 py-3 font-medium text-white">{d.domain}</td>
                      <td className="px-4 py-3 font-mono text-xs text-neutral-400">{appNames[d.app_id] ?? d.app_id}</td>
                      <td className="px-4 py-3" colSpan={2}>
                        <DomainStatus domain={d} onCheck={() => handleCheck(d)} checking={checkingId === d.id} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </Shell>
  );
}
