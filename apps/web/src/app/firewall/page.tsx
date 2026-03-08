"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import {
  type NetworkPolicyRule,
  type NetworkPolicyDirection,
  type NetworkPolicyAction,
  type NetworkPolicyConfig,
  type DeployApp,
} from "@/lib/api";
import { getApi, isDemoMode } from "@/lib/get-api";
import { useState, useMemo, useCallback } from "react";
import {
  Wifi,
  Plus,
  Trash2,
  Pencil,
  ToggleLeft,
  ToggleRight,
  X,
  ArrowDownLeft,
  ArrowUpRight,
} from "lucide-react";

function configSummary(rule: NetworkPolicyRule): string {
  const parts: string[] = [];
  const c = rule.config;
  if (c.cidrs?.length) parts.push(`CIDRs: ${c.cidrs.join(", ")}`);
  if (c.ports?.length)
    parts.push(`Ports: ${c.ports.map((p) => `${p.port}/${p.protocol}`).join(", ")}`);
  if (c.namespaces?.length) parts.push(`NS: ${c.namespaces.join(", ")}`);
  if (c.fqdns?.length) parts.push(`FQDN: ${c.fqdns.join(", ")}`);
  if (c.pod_labels && Object.keys(c.pod_labels).length > 0)
    parts.push(
      `Labels: ${Object.entries(c.pod_labels).map(([k, v]) => `${k}=${v}`).join(", ")}`
    );
  return parts.join(" | ") || "No filters";
}

interface RuleForm {
  name: string;
  direction: NetworkPolicyDirection;
  action: NetworkPolicyAction;
  priority: number;
  enabled: boolean;
  config: NetworkPolicyConfig;
}

const EMPTY_FORM: RuleForm = {
  name: "",
  direction: "ingress",
  action: "allow",
  priority: 0,
  enabled: true,
  config: { cidrs: [] },
};

export default function FirewallPage() {
  const { appsDeploy, networkPolicies } = getApi();
  const demo = isDemoMode();

  const [selectedAppId, setSelectedAppId] = useState<string>("");
  const [showModal, setShowModal] = useState(false);
  const [editingRule, setEditingRule] = useState<NetworkPolicyRule | null>(null);
  const [form, setForm] = useState<RuleForm>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const { data: appsData, loading: appsLoading } = useApi(() => appsDeploy.list(), []);
  const appList: DeployApp[] = useMemo(() => appsData?.items ?? [], [appsData]);
  const effectiveAppId = selectedAppId || appList[0]?.id || "";

  const {
    data: rulesData,
    loading: rulesLoading,
    error: rulesError,
    refetch,
  } = useApi(
    () =>
      effectiveAppId
        ? networkPolicies.listRules(effectiveAppId)
        : Promise.resolve({ rules: [], total: 0 }),
    [effectiveAppId]
  );

  const rules: NetworkPolicyRule[] = useMemo(
    () => (rulesData?.rules ?? []).sort((a, b) => a.priority - b.priority),
    [rulesData]
  );

  const handleCreate = useCallback(async () => {
    if (!effectiveAppId || !form.name.trim()) return;
    setSaving(true);
    try {
      await networkPolicies.createRule(effectiveAppId, {
        name: form.name.trim(),
        direction: form.direction,
        action: form.action,
        enabled: form.enabled,
        priority: form.priority,
        config: form.config,
      });
      setShowModal(false);
      setForm(EMPTY_FORM);
      refetch();
    } catch {
      // silent
    } finally {
      setSaving(false);
    }
  }, [effectiveAppId, form, networkPolicies, refetch]);

  const handleUpdate = useCallback(async () => {
    if (!effectiveAppId || !editingRule) return;
    setSaving(true);
    try {
      await networkPolicies.updateRule(effectiveAppId, editingRule.id, {
        name: form.name.trim() || undefined,
        enabled: form.enabled,
        priority: form.priority,
        config: form.config,
      });
      setEditingRule(null);
      setForm(EMPTY_FORM);
      refetch();
    } catch {
      // silent
    } finally {
      setSaving(false);
    }
  }, [effectiveAppId, editingRule, form, networkPolicies, refetch]);

  const handleDelete = useCallback(
    async (ruleId: string) => {
      if (!effectiveAppId || deletingId) return;
      setDeletingId(ruleId);
      try {
        await networkPolicies.deleteRule(effectiveAppId, ruleId);
        refetch();
      } catch {
        // silent
      } finally {
        setDeletingId(null);
      }
    },
    [effectiveAppId, networkPolicies, refetch, deletingId]
  );

  const handleToggle = useCallback(
    async (rule: NetworkPolicyRule) => {
      if (!effectiveAppId || togglingId) return;
      setTogglingId(rule.id);
      try {
        await networkPolicies.updateRule(effectiveAppId, rule.id, {
          enabled: !rule.enabled,
        });
        refetch();
      } catch {
        // silent
      } finally {
        setTogglingId(null);
      }
    },
    [effectiveAppId, networkPolicies, refetch, togglingId]
  );

  const openEdit = (rule: NetworkPolicyRule) => {
    setEditingRule(rule);
    setForm({
      name: rule.name,
      direction: rule.direction,
      action: rule.action,
      priority: rule.priority,
      enabled: rule.enabled,
      config: { ...rule.config },
    });
  };

  const loading = appsLoading || rulesLoading;

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={6} rows={4} />
      </Shell>
    );
  }

  if (rulesError) {
    return (
      <Shell>
        <ErrorState message={rulesError} onRetry={refetch} />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2">
              <Wifi className="h-5 w-5 text-accent-400" />
              <h1 className="text-lg font-semibold text-white">
                Network Policies
              </h1>
            </div>
            <p className="mt-1 text-sm text-neutral-500">
              Cilium-based network firewall rules per app (Business+ only)
            </p>
          </div>
          {!demo && effectiveAppId && (
            <button
              onClick={() => {
                setForm(EMPTY_FORM);
                setShowModal(true);
              }}
              className="flex items-center gap-1.5 rounded-lg bg-accent-500 px-3 py-2 text-sm font-medium text-white hover:bg-accent-400 transition-colors"
            >
              <Plus className="h-4 w-4" />
              Add Rule
            </button>
          )}
        </div>

        {/* App Selector */}
        {appList.length > 0 && (
          <div className="flex items-center gap-3">
            <label className="text-sm text-neutral-400">App:</label>
            <select
              value={effectiveAppId}
              onChange={(e) => setSelectedAppId(e.target.value)}
              className="rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
            >
              {appList.map((app) => (
                <option key={app.id} value={app.id}>
                  {app.name}
                </option>
              ))}
            </select>
            <span className="text-xs text-neutral-600">
              {rules.length} rule{rules.length !== 1 ? "s" : ""}
            </span>
          </div>
        )}

        {/* Rules Table */}
        {rules.length === 0 ? (
          <EmptyState
            title="No network policies"
            description={
              effectiveAppId
                ? "No firewall rules configured for this app yet."
                : "Select an app to manage its network policies."
            }
          />
        ) : (
          <div className="overflow-hidden rounded-xl border border-border">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border bg-surface-100 text-left text-xs font-medium text-neutral-500">
                    <th className="px-4 py-3 w-10">Pri</th>
                    <th className="px-4 py-3">Name</th>
                    <th className="px-4 py-3 w-24">Direction</th>
                    <th className="px-4 py-3 w-20">Action</th>
                    <th className="px-4 py-3">Filters</th>
                    <th className="px-4 py-3 w-20">Status</th>
                    <th className="px-4 py-3 w-28">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {rules.map((rule) => (
                    <tr
                      key={rule.id}
                      className={`bg-surface-50 transition-colors hover:bg-surface-100 ${!rule.enabled ? "opacity-50" : ""}`}
                    >
                      <td className="px-4 py-3 text-sm text-neutral-400 font-mono">
                        {rule.priority}
                      </td>
                      <td className="px-4 py-3 text-sm text-neutral-200">
                        {rule.name}
                      </td>
                      <td className="px-4 py-3">
                        <span className="flex items-center gap-1 text-xs text-neutral-300">
                          {rule.direction === "ingress" ? (
                            <ArrowDownLeft className="h-3.5 w-3.5 text-blue-400" />
                          ) : (
                            <ArrowUpRight className="h-3.5 w-3.5 text-orange-400" />
                          )}
                          {rule.direction}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${
                            rule.action === "allow"
                              ? "bg-green-500/15 text-green-400"
                              : "bg-red-500/15 text-red-400"
                          }`}
                        >
                          {rule.action}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="text-xs text-neutral-400 font-mono">
                          {configSummary(rule)}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${
                            rule.enabled
                              ? "bg-green-500/15 text-green-400"
                              : "bg-neutral-500/15 text-neutral-400"
                          }`}
                        >
                          {rule.enabled ? "Active" : "Disabled"}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1">
                          {!demo && (
                            <>
                              <button
                                onClick={() => handleToggle(rule)}
                                disabled={togglingId === rule.id}
                                className="rounded p-1.5 text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors disabled:opacity-50"
                                title={rule.enabled ? "Disable" : "Enable"}
                              >
                                {rule.enabled ? (
                                  <ToggleRight className="h-4 w-4 text-green-400" />
                                ) : (
                                  <ToggleLeft className="h-4 w-4" />
                                )}
                              </button>
                              <button
                                onClick={() => openEdit(rule)}
                                className="rounded p-1.5 text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors"
                                title="Edit"
                              >
                                <Pencil className="h-3.5 w-3.5" />
                              </button>
                              <button
                                onClick={() => handleDelete(rule.id)}
                                disabled={deletingId === rule.id}
                                className="rounded p-1.5 text-neutral-400 hover:text-red-400 hover:bg-surface-300 transition-colors disabled:opacity-50"
                                title="Delete"
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Create / Edit Modal */}
      {(showModal || editingRule) && (
        <PolicyModal
          title={editingRule ? "Edit Network Policy" : "Create Network Policy"}
          form={form}
          setForm={setForm}
          saving={saving}
          isEdit={!!editingRule}
          onSave={editingRule ? handleUpdate : handleCreate}
          onClose={() => {
            setShowModal(false);
            setEditingRule(null);
            setForm(EMPTY_FORM);
          }}
        />
      )}
    </Shell>
  );
}

function PolicyModal({
  title,
  form,
  setForm,
  saving,
  isEdit,
  onSave,
  onClose,
}: {
  title: string;
  form: RuleForm;
  setForm: (f: RuleForm) => void;
  saving: boolean;
  isEdit: boolean;
  onSave: () => void;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-lg rounded-xl border border-border bg-surface-50 p-6 shadow-xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">{title}</h2>
          <button
            onClick={onClose}
            className="rounded p-1 text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-neutral-300 mb-1">
              Name
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="e.g. Allow internal traffic"
              className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
            />
          </div>

          {/* Direction + Action */}
          {!isEdit && (
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-1">
                  Direction
                </label>
                <select
                  value={form.direction}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      direction: e.target.value as NetworkPolicyDirection,
                    })
                  }
                  className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                >
                  <option value="ingress">Ingress (incoming)</option>
                  <option value="egress">Egress (outgoing)</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-1">
                  Action
                </label>
                <select
                  value={form.action}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      action: e.target.value as NetworkPolicyAction,
                    })
                  }
                  className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                >
                  <option value="allow">Allow</option>
                  <option value="deny">Deny</option>
                </select>
              </div>
            </div>
          )}

          {/* Priority */}
          <div>
            <label className="block text-sm font-medium text-neutral-300 mb-1">
              Priority
            </label>
            <input
              type="number"
              value={form.priority}
              onChange={(e) =>
                setForm({ ...form, priority: parseInt(e.target.value) || 0 })
              }
              className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
            />
            <p className="mt-0.5 text-xs text-neutral-600">
              Lower numbers are evaluated first
            </p>
          </div>

          {/* Enabled */}
          <label className="flex items-center gap-2 text-sm text-neutral-300 cursor-pointer">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
              className="rounded border-border bg-surface-100 text-accent-500 focus:ring-accent-500"
            />
            Enabled
          </label>

          {/* Config */}
          <div className="border-t border-border pt-4">
            <p className="text-xs font-medium text-neutral-500 mb-3 uppercase tracking-wider">
              Filters
            </p>

            {/* CIDRs */}
            <div className="mb-3">
              <label className="block text-xs text-neutral-400 mb-1">
                CIDRs (one per line)
              </label>
              <textarea
                value={(form.config.cidrs ?? []).join("\n")}
                onChange={(e) =>
                  setForm({
                    ...form,
                    config: {
                      ...form.config,
                      cidrs: e.target.value
                        .split("\n")
                        .filter(Boolean),
                    },
                  })
                }
                rows={3}
                placeholder="10.0.0.0/8&#10;192.168.0.0/16"
                className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>

            {/* Ports */}
            <div className="mb-3">
              <label className="block text-xs text-neutral-400 mb-1">
                Ports (comma separated, e.g. 80/TCP, 443/TCP, 53/UDP)
              </label>
              <input
                type="text"
                value={(form.config.ports ?? [])
                  .map((p) => `${p.port}/${p.protocol}`)
                  .join(", ")}
                onChange={(e) =>
                  setForm({
                    ...form,
                    config: {
                      ...form.config,
                      ports: e.target.value
                        .split(",")
                        .map((s) => s.trim())
                        .filter(Boolean)
                        .map((s) => {
                          const [port, proto] = s.split("/");
                          return {
                            port: parseInt(port) || 0,
                            protocol: (proto?.toUpperCase() === "UDP"
                              ? "UDP"
                              : "TCP") as "TCP" | "UDP",
                          };
                        }),
                    },
                  })
                }
                placeholder="80/TCP, 443/TCP"
                className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>

            {/* FQDNs */}
            <div className="mb-3">
              <label className="block text-xs text-neutral-400 mb-1">
                FQDNs (one per line, for egress)
              </label>
              <textarea
                value={(form.config.fqdns ?? []).join("\n")}
                onChange={(e) =>
                  setForm({
                    ...form,
                    config: {
                      ...form.config,
                      fqdns: e.target.value
                        .split("\n")
                        .filter(Boolean),
                    },
                  })
                }
                rows={2}
                placeholder="api.stripe.com&#10;*.googleapis.com"
                className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>

            {/* Namespaces */}
            <div>
              <label className="block text-xs text-neutral-400 mb-1">
                Namespaces (comma separated)
              </label>
              <input
                type="text"
                value={(form.config.namespaces ?? []).join(", ")}
                onChange={(e) =>
                  setForm({
                    ...form,
                    config: {
                      ...form.config,
                      namespaces: e.target.value
                        .split(",")
                        .map((s) => s.trim())
                        .filter(Boolean),
                    },
                  })
                }
                placeholder="zenith-databases, zenith-apps"
                className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="mt-6 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white hover:bg-surface-300 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onSave}
            disabled={saving || !form.name.trim()}
            className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? "Saving..." : isEdit ? "Update Rule" : "Create Rule"}
          </button>
        </div>
      </div>
    </div>
  );
}
