"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { Check, ChevronDown, ChevronRight, GitBranch, RotateCcw, Rocket } from "lucide-react";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useToast } from "@/components/toast";
import { useApi } from "@/hooks/use-api";
import { useDeployEvents } from "@/hooks/use-deploy-events";
import type { DeployApp, Deployment, Release } from "@/lib/api";
import { getApi } from "@/lib/get-api";

const TH = "whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500";

export function DeploymentsCard({ app }: { app: DeployApp }) {
  const appId = app.id;
  const { toast } = useToast();
  const { data, loading, error, refetch } = useApi(
    () => getApi().appsDeploy.listDeployments(appId),
    [appId]
  );
  const [showReleases, setShowReleases] = useState(false);
  const [releaseCount, setReleaseCount] = useState<number | null>(null);

  const handleDeployEvent = useCallback(
    (event: { app_id: string }) => {
      if (event.app_id === appId) refetch();
    },
    [appId, refetch]
  );
  useDeployEvents(handleDeployEvent);

  const handleRollback = async (deployId: string) => {
    try {
      await getApi().appsDeploy.rollback(appId, deployId);
      refetch();
    } catch {
      toast("error", "Failed to rollback deployment");
    }
  };

  const items = data?.items || [];

  return (
    <div className="rounded-lg border border-border bg-surface-100 p-5 md:col-span-2">
      <h3 className="mb-4 text-sm font-medium text-white">Recent Deployments</h3>
      {loading ? (
        <PageWithTableSkeleton cols={5} rows={3} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : items.length === 0 ? (
        <EmptyState title="No deployments yet" description="Push to your repository to trigger a deployment." />
      ) : (
        <div className="overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className={TH}>ID</th>
                <th className={TH}>Status</th>
                <th className={TH}>Git SHA</th>
                <th className={TH}>Created</th>
                <th className={TH}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((d: Deployment) => (
                <tr key={d.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">{d.id.slice(0, 8)}</td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <StatusBadge status={d.status as "running" | "deploying" | "stopped" | "crashed"} />
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                    {d.git_sha?.slice(0, 8) || "—"}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">
                    {new Date(d.created_at).toLocaleString()}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    {d.status !== "active" && (
                      <button
                        onClick={() => handleRollback(d.id)}
                        className="flex items-center gap-1 rounded px-2 py-1 text-xs text-neutral-400 hover:bg-surface-300 hover:text-white transition-colors"
                        title="Rollback to this deployment"
                      >
                        <RotateCcw className="h-3 w-3" />
                        Rollback
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-4">
        <button
          onClick={() => setShowReleases((v) => !v)}
          aria-expanded={showReleases}
          className="flex items-center gap-1 text-xs font-medium text-neutral-400 hover:text-white transition-colors"
        >
          {showReleases ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          {releaseCount === null ? "Show releases" : `Releases (${releaseCount})`}
        </button>
        {showReleases && <ReleasesSection appId={appId} onCount={setReleaseCount} onDeployed={refetch} />}
      </div>
    </div>
  );
}

function ReleasesSection({
  appId,
  onCount,
  onDeployed,
}: {
  appId: string;
  onCount: (n: number) => void;
  onDeployed: () => void;
}) {
  const { toast } = useToast();
  const { data, loading, error, refetch } = useApi(async () => {
    const res = await getApi().appsDeploy.listReleases(appId);
    onCount(res?.releases?.length ?? 0);
    return res;
  }, [appId]);
  const [deployingId, setDeployingId] = useState<string | null>(null);
  const [deployedId, setDeployedId] = useState<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => () => {
    if (timerRef.current) clearTimeout(timerRef.current);
  }, []);

  if (loading) return <div className="mt-3"><PageWithTableSkeleton cols={5} rows={3} /></div>;
  if (error) return <div className="mt-3"><ErrorState message={error} onRetry={refetch} /></div>;

  const releases = data?.releases || [];
  if (releases.length === 0) {
    return (
      <div className="mt-3">
        <EmptyState
          title="No releases yet"
          description="Push an image using zenith-actions in your CI pipeline to register releases here."
        />
      </div>
    );
  }

  const handleDeploy = async (releaseId: string) => {
    setDeployingId(releaseId);
    setDeployedId(null);
    try {
      await getApi().appsDeploy.deployRelease(appId, releaseId);
      setDeployedId(releaseId);
      onDeployed();
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => setDeployedId(null), 3000);
    } catch {
      toast("error", "Failed to deploy release");
    } finally {
      setDeployingId(null);
    }
  };

  return (
    <div className="mt-3 space-y-3">
      <p className="text-xs text-neutral-500">
        Image versions pushed by your CI pipeline. Click Deploy to roll out a specific version.
      </p>
      <div className="overflow-x-auto rounded-lg border border-border">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-border bg-surface-100">
              <th className={TH}>Image</th>
              <th className={TH}>Git SHA</th>
              <th className={TH}>Branch</th>
              <th className={TH}>Message</th>
              <th className={TH}>Date</th>
              <th className={`${TH} w-28`}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {releases.map((rel: Release, idx: number) => (
              <tr
                key={rel.id}
                className={`border-b border-border last:border-0 hover:bg-surface-200 transition-colors ${idx === 0 ? "bg-surface-100/50" : ""}`}
              >
                <td className="whitespace-nowrap px-4 py-3">
                  <div className="flex items-center gap-2">
                    {idx === 0 && (
                      <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-400">
                        latest
                      </span>
                    )}
                    <code className="text-xs text-neutral-300">{rel.image.split(":").pop() || rel.image}</code>
                  </div>
                </td>
                <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                  {rel.git_sha?.slice(0, 8) || "—"}
                </td>
                <td className="whitespace-nowrap px-4 py-3">
                  <span className="flex items-center gap-1 text-xs text-neutral-400">
                    <GitBranch className="h-3 w-3" />
                    {rel.branch || "main"}
                  </span>
                </td>
                <td className="max-w-xs truncate px-4 py-3 text-xs text-neutral-300" title={rel.message}>
                  {rel.message || "—"}
                </td>
                <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-500">
                  {new Date(rel.created_at).toLocaleString()}
                </td>
                <td className="whitespace-nowrap px-4 py-3">
                  {deployedId === rel.id ? (
                    <span className="flex items-center gap-1 text-xs font-medium text-emerald-400">
                      <Check className="h-3 w-3" />
                      Triggered
                    </span>
                  ) : (
                    <button
                      onClick={() => handleDeploy(rel.id)}
                      disabled={deployingId !== null}
                      className="flex items-center gap-1 rounded-md bg-accent-500/10 px-3 py-1.5 text-xs font-medium text-accent-400 hover:bg-accent-500/20 disabled:opacity-50 transition-colors"
                    >
                      {deployingId === rel.id ? (
                        <div className="h-3 w-3 animate-spin rounded-full border border-accent-400 border-t-transparent" />
                      ) : (
                        <Rocket className="h-3 w-3" />
                      )}
                      Deploy
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
