"use client";
import { useState } from "react";
import { Globe, Plus, Trash2 } from "lucide-react";
import { SHOW_PLANS, customDomainInstruction } from "@/lib/edition";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useToast } from "@/components/toast";
import { useApi } from "@/hooks/use-api";
import type { DeployApp, CustomDomain } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { DomainStatus } from "@/components/domain-status";

// Custom domains, web apps only (half width grid cell).
export function DomainsCard({ app }: { app: DeployApp }) {
  if (app.app_type && app.app_type !== "web") return null;
  return <DomainsBody appId={app.id} />;
}

function DomainsBody({ appId }: { appId: string }) {
  const { toast } = useToast();
  const { data: domains, loading, error, refetch } = useApi(
    async () => getApi().appsDeploy.listDomains(appId),
    [appId]
  );

  const [showAdd, setShowAdd] = useState(false);
  const [domainInput, setDomainInput] = useState("");
  const [adding, setAdding] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [checkingId, setCheckingId] = useState<string | null>(null);


  const items: CustomDomain[] = domains || [];

  const handleCheck = async (domainId: string) => {
    setCheckingId(domainId);
    try {
      await getApi().appsDeploy.verifyDomain(appId, domainId);
      refetch();
    } catch {
      toast("error", "Domain check failed");
    } finally {
      setCheckingId(null);
    }
  };

  const handleAdd = async () => {
    if (!domainInput.trim()) return;
    setAdding(true);
    try {
      await getApi().appsDeploy.addDomain(appId, domainInput.trim());
      setShowAdd(false);
      setDomainInput("");
      refetch();
    } catch {
      toast("error", "Failed to add custom domain");
    } finally {
      setAdding(false);
    }
  };

  const handleDelete = async (domainId: string) => {
    setDeletingId(domainId);
    try {
      await getApi().appsDeploy.deleteDomain(appId, domainId);
      refetch();
    } catch {
      toast("error", "Failed to delete custom domain");
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <section className="rounded-lg border border-border bg-surface-100 p-5">
      <h2 className="mb-4 text-sm font-medium text-white">Domains</h2>
      {loading ? (
        <PageWithTableSkeleton cols={4} rows={2} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs text-neutral-500">
            {SHOW_PLANS ? "Custom domains for your app. Add a CNAME record pointing to your app's subdomain." : "Custom domains for your app. Point your domain at this server with an A record."}
          </p>
          <p className="text-xs text-neutral-600 mt-1">
            {SHOW_PLANS && <>Requires <span className="text-amber-400">Pro</span> plan or higher.</>}
          </p>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Add Domain
        </button>
      </div>

      {showAdd && (
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h3 className="mb-2 text-sm font-medium text-neutral-400">Add Custom Domain</h3>
          <p className="mb-3 text-xs text-neutral-600">
            {SHOW_PLANS ? (
              <>First, add a CNAME record: <code className="text-neutral-400">your-domain.com → {appId}.freezenith.com</code></>
            ) : (
              customDomainInstruction()
            )}
          </p>
          <div className="flex items-end gap-3">
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-neutral-400">Domain</label>
              <input
                type="text"
                value={domainInput}
                onChange={(e) => setDomainInput(e.target.value.toLowerCase())}
                placeholder="app.example.com"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none font-mono"
              />
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setShowAdd(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleAdd}
                disabled={!domainInput.trim() || adding}
                className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 disabled:opacity-50 transition-colors"
              >
                {adding ? (
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                ) : (
                  <Globe className="h-4 w-4" />
                )}
                Add
              </button>
            </div>
          </div>
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState
          title="No custom domains"
          description="Add a custom domain to serve your app from your own URL."
        />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Domain</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500" colSpan={2}>Status</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Added</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500 w-20">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((d) => (
                <tr key={d.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-sm text-accent-400">
                    {d.domain}
                  </td>
                  <td className="px-4 py-3" colSpan={2}>
                    <DomainStatus domain={d} onCheck={() => handleCheck(d.id)} checking={checkingId === d.id} />
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                    {new Date(d.created_at).toLocaleDateString()}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <button
                      onClick={() => handleDelete(d.id)}
                      disabled={deletingId === d.id}
                      className="rounded p-1 text-neutral-500 hover:bg-red-500/10 hover:text-red-400 transition-colors disabled:opacity-50"
                      title="Remove domain"
                    >
                      {deletingId === d.id ? (
                        <div className="h-3.5 w-3.5 animate-spin rounded-full border border-red-400 border-t-transparent" />
                      ) : (
                        <Trash2 className="h-3.5 w-3.5" />
                      )}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
      )}
    </section>
  );
}
