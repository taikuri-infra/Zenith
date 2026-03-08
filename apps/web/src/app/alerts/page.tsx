"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { type AlertRule, type AlertSeverity, type CustomMetric, type DeployApp } from "@/lib/api";
import { getApi, isDemoMode } from "@/lib/get-api";
import { useState, useMemo, useCallback } from "react";
import {
  Bell,
  Plus,
  Trash2,
  Pencil,
  ToggleLeft,
  ToggleRight,
  X,
  Mail,
  Hash,
} from "lucide-react";

const SEVERITY_COLORS: Record<string, string> = {
  critical: "bg-red-500/15 text-red-400",
  warning: "bg-yellow-500/15 text-yellow-400",
  info: "bg-blue-500/15 text-blue-400",
};

export default function AlertsPage() {
  const { appsDeploy, alerts: alertsApi } = getApi();
  const demo = isDemoMode();
  const [selectedAppId, setSelectedAppId] = useState<string>("");
  const [tab, setTab] = useState<"rules" | "metrics">("rules");
  const [showModal, setShowModal] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const { data: appsData, loading: appsLoading } = useApi(() => appsDeploy.list(), []);
  const appList: DeployApp[] = useMemo(() => appsData?.items ?? [], [appsData]);
  const effectiveAppId = selectedAppId || appList[0]?.id || "";

  const { data: rulesData, loading: rulesLoading, error: rulesError, refetch: refetchRules } = useApi(
    () => effectiveAppId ? alertsApi.listRules(effectiveAppId) : Promise.resolve({ rules: [], total: 0 }),
    [effectiveAppId]
  );

  const { data: metricsData, loading: metricsLoading, refetch: refetchMetrics } = useApi(
    () => effectiveAppId ? alertsApi.listMetrics(effectiveAppId) : Promise.resolve({ metrics: [], total: 0 }),
    [effectiveAppId]
  );

  const rules: AlertRule[] = useMemo(() => rulesData?.rules ?? [], [rulesData]);
  const metrics: CustomMetric[] = useMemo(() => metricsData?.metrics ?? [], [metricsData]);

  const handleToggle = useCallback(async (rule: AlertRule) => {
    if (!effectiveAppId || togglingId) return;
    setTogglingId(rule.id);
    try {
      await alertsApi.updateRule(effectiveAppId, rule.id, { enabled: !rule.enabled });
      refetchRules();
    } catch { /* silent */ } finally {
      setTogglingId(null);
    }
  }, [effectiveAppId, alertsApi, refetchRules, togglingId]);

  const handleDeleteRule = useCallback(async (ruleId: string) => {
    if (!effectiveAppId || deletingId) return;
    setDeletingId(ruleId);
    try {
      await alertsApi.deleteRule(effectiveAppId, ruleId);
      refetchRules();
    } catch { /* silent */ } finally {
      setDeletingId(null);
    }
  }, [effectiveAppId, alertsApi, refetchRules, deletingId]);

  const handleDeleteMetric = useCallback(async (metricId: string) => {
    if (!effectiveAppId || deletingId) return;
    setDeletingId(metricId);
    try {
      await alertsApi.deleteMetric(effectiveAppId, metricId);
      refetchMetrics();
    } catch { /* silent */ } finally {
      setDeletingId(null);
    }
  }, [effectiveAppId, alertsApi, refetchMetrics, deletingId]);

  // Rule form state
  const [ruleForm, setRuleForm] = useState<{ name: string; metric: string; condition: string; duration: string; severity: AlertSeverity; description: string; notify_email: boolean; notify_slack: boolean }>({ name: "", metric: "", condition: "", duration: "5m", severity: "warning", description: "", notify_email: true, notify_slack: false });
  const [metricForm, setMetricForm] = useState({ name: "", expression: "" });

  const handleCreateRule = useCallback(async () => {
    if (!effectiveAppId || !ruleForm.name || !ruleForm.metric || !ruleForm.condition) return;
    setSaving(true);
    try {
      await alertsApi.createRule(effectiveAppId, ruleForm);
      setShowModal(false);
      setRuleForm({ name: "", metric: "", condition: "", duration: "5m", severity: "warning", description: "", notify_email: true, notify_slack: false });
      refetchRules();
    } catch { /* silent */ } finally {
      setSaving(false);
    }
  }, [effectiveAppId, ruleForm, alertsApi, refetchRules]);

  const handleCreateMetric = useCallback(async () => {
    if (!effectiveAppId || !metricForm.name || !metricForm.expression) return;
    setSaving(true);
    try {
      await alertsApi.createMetric(effectiveAppId, metricForm);
      setShowModal(false);
      setMetricForm({ name: "", expression: "" });
      refetchMetrics();
    } catch { /* silent */ } finally {
      setSaving(false);
    }
  }, [effectiveAppId, metricForm, alertsApi, refetchMetrics]);

  const loading = appsLoading || rulesLoading || metricsLoading;

  if (loading) return <Shell><PageWithTableSkeleton cols={6} rows={4} /></Shell>;
  if (rulesError) return <Shell><ErrorState message={rulesError} onRetry={refetchRules} /></Shell>;

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2">
              <Bell className="h-5 w-5 text-accent-400" />
              <h1 className="text-lg font-semibold text-white">Alerts & Custom Metrics</h1>
            </div>
            <p className="mt-1 text-sm text-neutral-500">Custom alerting rules and Prometheus recording rules (Business+ only)</p>
          </div>
          {!demo && effectiveAppId && (
            <button onClick={() => setShowModal(true)} className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-2 text-sm font-medium text-white hover:bg-accent-400 transition-colors">
              <Plus className="h-4 w-4" />
              {tab === "rules" ? "Add Alert" : "Add Metric"}
            </button>
          )}
        </div>

        {appList.length > 0 && (
          <div className="flex items-center gap-3">
            <label className="text-sm text-neutral-400">App:</label>
            <select value={effectiveAppId} onChange={(e) => setSelectedAppId(e.target.value)} className="rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none">
              {appList.map((app) => <option key={app.id} value={app.id}>{app.name}</option>)}
            </select>
          </div>
        )}

        {/* Tabs */}
        <div className="flex gap-1 rounded-lg bg-surface-100 p-1 w-fit">
          <button onClick={() => setTab("rules")} className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${tab === "rules" ? "bg-surface-300 text-white" : "text-neutral-400 hover:text-white"}`}>
            Alert Rules ({rules.length})
          </button>
          <button onClick={() => setTab("metrics")} className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${tab === "metrics" ? "bg-surface-300 text-white" : "text-neutral-400 hover:text-white"}`}>
            Custom Metrics ({metrics.length})
          </button>
        </div>

        {/* Alert Rules Table */}
        {tab === "rules" && (
          rules.length === 0 ? (
            <EmptyState title="No alert rules" description="Create your first alert rule to get notified when metrics exceed thresholds." />
          ) : (
            <div className="overflow-hidden rounded-xl border border-border">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-border bg-surface-100 text-left text-xs font-medium text-neutral-500">
                      <th className="px-4 py-3">Name</th>
                      <th className="px-4 py-3">Metric</th>
                      <th className="px-4 py-3">Condition</th>
                      <th className="px-4 py-3">Severity</th>
                      <th className="px-4 py-3">Notify</th>
                      <th className="px-4 py-3 w-20">Status</th>
                      <th className="px-4 py-3 w-28">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {rules.map((rule) => (
                      <tr key={rule.id} className={`bg-surface-50 transition-colors hover:bg-surface-100 ${!rule.enabled ? "opacity-50" : ""}`}>
                        <td className="px-4 py-3 text-sm text-neutral-200">{rule.name}</td>
                        <td className="px-4 py-3"><code className="text-xs text-neutral-400 font-mono">{rule.metric}</code></td>
                        <td className="px-4 py-3"><code className="text-xs text-neutral-400 font-mono">{rule.condition} for {rule.duration}</code></td>
                        <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${SEVERITY_COLORS[rule.severity] || ""}`}>{rule.severity}</span></td>
                        <td className="px-4 py-3">
                          <div className="flex gap-1">
                            {rule.notify_email && <span title="Email"><Mail className="h-3.5 w-3.5 text-blue-400" /></span>}
                            {rule.notify_slack && <span title="Slack"><Hash className="h-3.5 w-3.5 text-purple-400" /></span>}
                          </div>
                        </td>
                        <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${rule.enabled ? "bg-green-500/15 text-green-400" : "bg-neutral-500/15 text-neutral-400"}`}>{rule.enabled ? "Active" : "Disabled"}</span></td>
                        <td className="px-4 py-3">
                          {!demo && (
                            <div className="flex items-center gap-1">
                              <button onClick={() => handleToggle(rule)} disabled={togglingId === rule.id} className="rounded p-1.5 text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors disabled:opacity-50" title={rule.enabled ? "Disable" : "Enable"}>
                                {rule.enabled ? <ToggleRight className="h-4 w-4 text-green-400" /> : <ToggleLeft className="h-4 w-4" />}
                              </button>
                              <button onClick={() => handleDeleteRule(rule.id)} disabled={deletingId === rule.id} className="rounded p-1.5 text-neutral-400 hover:text-red-400 hover:bg-surface-300 transition-colors disabled:opacity-50" title="Delete">
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )
        )}

        {/* Custom Metrics Table */}
        {tab === "metrics" && (
          metrics.length === 0 ? (
            <EmptyState title="No custom metrics" description="Create custom Prometheus recording rules for derived metrics." />
          ) : (
            <div className="overflow-hidden rounded-xl border border-border">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-border bg-surface-100 text-left text-xs font-medium text-neutral-500">
                      <th className="px-4 py-3">Name</th>
                      <th className="px-4 py-3">PromQL Expression</th>
                      <th className="px-4 py-3 w-28">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {metrics.map((m) => (
                      <tr key={m.id} className="bg-surface-50 transition-colors hover:bg-surface-100">
                        <td className="px-4 py-3 text-sm text-neutral-200">{m.name}</td>
                        <td className="px-4 py-3"><code className="text-xs text-neutral-400 font-mono break-all">{m.expression}</code></td>
                        <td className="px-4 py-3">
                          {!demo && (
                            <button onClick={() => handleDeleteMetric(m.id)} disabled={deletingId === m.id} className="rounded p-1.5 text-neutral-400 hover:text-red-400 hover:bg-surface-300 transition-colors disabled:opacity-50" title="Delete">
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )
        )}
      </div>

      {/* Create Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="w-full max-w-lg rounded-xl border border-border bg-surface-50 p-6 shadow-xl">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-white">{tab === "rules" ? "Create Alert Rule" : "Create Custom Metric"}</h2>
              <button onClick={() => setShowModal(false)} className="rounded p-1 text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors"><X className="h-5 w-5" /></button>
            </div>

            {tab === "rules" ? (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">Name</label>
                  <input type="text" value={ruleForm.name} onChange={(e) => setRuleForm({ ...ruleForm, name: e.target.value })} placeholder="e.g. High CPU Usage" className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">Metric (PromQL)</label>
                  <input type="text" value={ruleForm.metric} onChange={(e) => setRuleForm({ ...ruleForm, metric: e.target.value })} placeholder="container_cpu_usage_seconds_total" className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-sm font-medium text-neutral-300 mb-1">Condition</label>
                    <input type="text" value={ruleForm.condition} onChange={(e) => setRuleForm({ ...ruleForm, condition: e.target.value })} placeholder="> 0.8" className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-neutral-300 mb-1">Duration</label>
                    <input type="text" value={ruleForm.duration} onChange={(e) => setRuleForm({ ...ruleForm, duration: e.target.value })} placeholder="5m" className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">Severity</label>
                  <select value={ruleForm.severity} onChange={(e) => setRuleForm({ ...ruleForm, severity: e.target.value as "critical" | "warning" | "info" })} className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none">
                    <option value="critical">Critical</option>
                    <option value="warning">Warning</option>
                    <option value="info">Info</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">Description</label>
                  <input type="text" value={ruleForm.description} onChange={(e) => setRuleForm({ ...ruleForm, description: e.target.value })} placeholder="Optional description" className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                </div>
                <div className="flex gap-4">
                  <label className="flex items-center gap-2 text-sm text-neutral-300 cursor-pointer">
                    <input type="checkbox" checked={ruleForm.notify_email} onChange={(e) => setRuleForm({ ...ruleForm, notify_email: e.target.checked })} className="rounded border-border bg-surface-100 text-accent-500" /> Email
                  </label>
                  <label className="flex items-center gap-2 text-sm text-neutral-300 cursor-pointer">
                    <input type="checkbox" checked={ruleForm.notify_slack} onChange={(e) => setRuleForm({ ...ruleForm, notify_slack: e.target.checked })} className="rounded border-border bg-surface-100 text-accent-500" /> Slack
                  </label>
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">Name</label>
                  <input type="text" value={metricForm.name} onChange={(e) => setMetricForm({ ...metricForm, name: e.target.value })} placeholder="app:request_rate:5m" className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-300 mb-1">PromQL Expression</label>
                  <textarea value={metricForm.expression} onChange={(e) => setMetricForm({ ...metricForm, expression: e.target.value })} rows={3} placeholder='rate(http_requests_total{app="my-app"}[5m])' className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                </div>
              </div>
            )}

            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowModal(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors">Cancel</button>
              <button onClick={tab === "rules" ? handleCreateRule : handleCreateMetric} disabled={saving} className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
                {saving ? "Saving..." : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}
    </Shell>
  );
}
