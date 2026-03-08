"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { type WAFRule, type WAFRuleType, type WAFConfig, type DeployApp } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import { isDemoMode } from "@/lib/get-api";
import { useState, useMemo, useCallback } from "react";
import {
  ShieldAlert,
  Plus,
  Trash2,
  Pencil,
  ToggleLeft,
  ToggleRight,
  X,
  Globe,
  Server,
  Zap,
  FileText,
} from "lucide-react";

const RULE_TYPE_LABELS: Record<WAFRuleType, string> = {
  rate_limit: "Rate Limit",
  ip_block: "IP Block",
  ip_allow: "IP Allow",
  body_limit: "Body Limit",
  geo_block: "Geo Block",
  header_rule: "Header Rule",
};

const RULE_TYPE_COLORS: Record<WAFRuleType, string> = {
  rate_limit: "bg-yellow-500/15 text-yellow-400",
  ip_block: "bg-red-500/15 text-red-400",
  ip_allow: "bg-green-500/15 text-green-400",
  body_limit: "bg-blue-500/15 text-blue-400",
  geo_block: "bg-purple-500/15 text-purple-400",
  header_rule: "bg-orange-500/15 text-orange-400",
};

const RULE_TYPE_ICONS: Record<WAFRuleType, typeof Globe> = {
  rate_limit: Zap,
  ip_block: Server,
  ip_allow: Server,
  body_limit: FileText,
  geo_block: Globe,
  header_rule: FileText,
};

interface CreateRuleForm {
  name: string;
  type: WAFRuleType;
  priority: number;
  enabled: boolean;
  config: WAFConfig;
}

const EMPTY_FORM: CreateRuleForm = {
  name: "",
  type: "rate_limit",
  priority: 0,
  enabled: true,
  config: { rate_per_second: 100, burst_size: 200 },
};

function configSummary(type: WAFRuleType, config: WAFConfig): string {
  switch (type) {
    case "rate_limit":
      return `${config.rate_per_second ?? 0} req/s, burst ${config.burst_size ?? 0}`;
    case "ip_block":
    case "ip_allow":
      return (config.ip_addresses ?? []).join(", ") || "No IPs";
    case "body_limit":
      return `Max ${config.max_body_size_kb ?? 0} KB`;
    case "geo_block":
      return (config.countries ?? []).join(", ") || "No countries";
    case "header_rule":
      return `${config.header_name}: /${config.header_match}/ → ${config.action}`;
  }
}

export default function WAFPage() {
  const { appsDeploy, waf } = getApi();
  const demo = isDemoMode();

  const [selectedAppId, setSelectedAppId] = useState<string>("");
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingRule, setEditingRule] = useState<WAFRule | null>(null);
  const [form, setForm] = useState<CreateRuleForm>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const { data: appsData, loading: appsLoading } = useApi(() => appsDeploy.list(), []);

  const appList: DeployApp[] = useMemo(() => appsData?.items ?? [], [appsData]);

  // Auto-select first app
  const effectiveAppId = selectedAppId || appList[0]?.id || "";

  const {
    data: rulesData,
    loading: rulesLoading,
    error: rulesError,
    refetch,
  } = useApi(
    () => (effectiveAppId ? waf.listRules(effectiveAppId) : Promise.resolve({ rules: [], total: 0 })),
    [effectiveAppId]
  );

  const rules: WAFRule[] = useMemo(
    () => (rulesData?.rules ?? []).sort((a, b) => a.priority - b.priority),
    [rulesData]
  );

  const handleCreate = useCallback(async () => {
    if (!effectiveAppId || !form.name.trim()) return;
    setSaving(true);
    try {
      await waf.createRule(effectiveAppId, {
        name: form.name.trim(),
        type: form.type,
        enabled: form.enabled,
        priority: form.priority,
        config: form.config,
      });
      setShowCreateModal(false);
      setForm(EMPTY_FORM);
      refetch();
    } catch {
      // silent
    } finally {
      setSaving(false);
    }
  }, [effectiveAppId, form, waf, refetch]);

  const handleUpdate = useCallback(async () => {
    if (!effectiveAppId || !editingRule) return;
    setSaving(true);
    try {
      await waf.updateRule(effectiveAppId, editingRule.id, {
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
  }, [effectiveAppId, editingRule, form, waf, refetch]);

  const handleDelete = useCallback(
    async (ruleId: string) => {
      if (!effectiveAppId || deletingId) return;
      setDeletingId(ruleId);
      try {
        await waf.deleteRule(effectiveAppId, ruleId);
        refetch();
      } catch {
        // silent
      } finally {
        setDeletingId(null);
      }
    },
    [effectiveAppId, waf, refetch, deletingId]
  );

  const handleToggle = useCallback(
    async (rule: WAFRule) => {
      if (!effectiveAppId || togglingId) return;
      setTogglingId(rule.id);
      try {
        await waf.updateRule(effectiveAppId, rule.id, { enabled: !rule.enabled });
        refetch();
      } catch {
        // silent
      } finally {
        setTogglingId(null);
      }
    },
    [effectiveAppId, waf, refetch, togglingId]
  );

  const openEdit = (rule: WAFRule) => {
    setEditingRule(rule);
    setForm({
      name: rule.name,
      type: rule.type,
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

  const selectedApp = appList.find((a) => a.id === effectiveAppId);

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2">
              <ShieldAlert className="h-5 w-5 text-accent-400" />
              <h1 className="text-lg font-semibold text-white">WAF Rules</h1>
            </div>
            <p className="mt-1 text-sm text-neutral-500">
              Web Application Firewall configuration per app (Business+ only)
            </p>
          </div>
          {!demo && effectiveAppId && (
            <button
              onClick={() => {
                setForm(EMPTY_FORM);
                setShowCreateModal(true);
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
            {selectedApp && (
              <span className="text-xs text-neutral-600">
                {rules.length} rule{rules.length !== 1 ? "s" : ""}
              </span>
            )}
          </div>
        )}

        {/* Rules Table */}
        {rules.length === 0 ? (
          <EmptyState
            title="No WAF rules"
            description={
              effectiveAppId
                ? "No firewall rules configured for this app yet."
                : "Select an app to manage its WAF rules."
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
                    <th className="px-4 py-3">Type</th>
                    <th className="px-4 py-3">Config</th>
                    <th className="px-4 py-3 w-20">Status</th>
                    <th className="px-4 py-3 w-28">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {rules.map((rule) => {
                    const TypeIcon = RULE_TYPE_ICONS[rule.type];
                    return (
                      <tr
                        key={rule.id}
                        className={`bg-surface-50 transition-colors hover:bg-surface-100 ${!rule.enabled ? "opacity-50" : ""}`}
                      >
                        <td className="px-4 py-3 text-sm text-neutral-400 font-mono">
                          {rule.priority}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <TypeIcon className="h-4 w-4 text-neutral-500 flex-shrink-0" />
                            <span className="text-sm text-neutral-200">{rule.name}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${RULE_TYPE_COLORS[rule.type]}`}
                          >
                            {RULE_TYPE_LABELS[rule.type]}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <span className="text-xs text-neutral-400 font-mono">
                            {configSummary(rule.type, rule.config)}
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
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Create / Edit Modal */}
      {(showCreateModal || editingRule) && (
        <RuleModal
          title={editingRule ? "Edit WAF Rule" : "Create WAF Rule"}
          form={form}
          setForm={setForm}
          saving={saving}
          isEdit={!!editingRule}
          onSave={editingRule ? handleUpdate : handleCreate}
          onClose={() => {
            setShowCreateModal(false);
            setEditingRule(null);
            setForm(EMPTY_FORM);
          }}
        />
      )}
    </Shell>
  );
}

function RuleModal({
  title,
  form,
  setForm,
  saving,
  isEdit,
  onSave,
  onClose,
}: {
  title: string;
  form: CreateRuleForm;
  setForm: (f: CreateRuleForm) => void;
  saving: boolean;
  isEdit: boolean;
  onSave: () => void;
  onClose: () => void;
}) {
  const updateConfig = (key: string, value: unknown) => {
    setForm({ ...form, config: { ...form.config, [key]: value } });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-full max-w-lg rounded-xl border border-border bg-surface-50 p-6 shadow-xl">
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
            <label className="block text-sm font-medium text-neutral-300 mb-1">Name</label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="e.g. Rate limit public API"
              className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
            />
          </div>

          {/* Type */}
          {!isEdit && (
            <div>
              <label className="block text-sm font-medium text-neutral-300 mb-1">Rule Type</label>
              <select
                value={form.type}
                onChange={(e) => {
                  const type = e.target.value as WAFRuleType;
                  const defaultConfigs: Record<WAFRuleType, WAFConfig> = {
                    rate_limit: { rate_per_second: 100, burst_size: 200 },
                    ip_block: { ip_addresses: [] },
                    ip_allow: { ip_addresses: [] },
                    body_limit: { max_body_size_kb: 1024 },
                    geo_block: { countries: [] },
                    header_rule: { header_name: "", header_match: "", action: "block" },
                  };
                  setForm({ ...form, type, config: defaultConfigs[type] });
                }}
                className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
              >
                {Object.entries(RULE_TYPE_LABELS).map(([k, v]) => (
                  <option key={k} value={k}>{v}</option>
                ))}
              </select>
            </div>
          )}

          {/* Priority */}
          <div>
            <label className="block text-sm font-medium text-neutral-300 mb-1">Priority</label>
            <input
              type="number"
              value={form.priority}
              onChange={(e) => setForm({ ...form, priority: parseInt(e.target.value) || 0 })}
              className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
            />
            <p className="mt-0.5 text-xs text-neutral-600">Lower numbers execute first</p>
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

          {/* Config fields per type */}
          <div className="border-t border-border pt-4">
            <p className="text-xs font-medium text-neutral-500 mb-3 uppercase tracking-wider">Configuration</p>
            {form.type === "rate_limit" && (
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs text-neutral-400 mb-1">Requests/sec</label>
                  <input
                    type="number"
                    value={form.config.rate_per_second ?? 100}
                    onChange={(e) => updateConfig("rate_per_second", parseInt(e.target.value) || 0)}
                    className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-neutral-400 mb-1">Burst size</label>
                  <input
                    type="number"
                    value={form.config.burst_size ?? 200}
                    onChange={(e) => updateConfig("burst_size", parseInt(e.target.value) || 0)}
                    className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                  />
                </div>
              </div>
            )}
            {(form.type === "ip_block" || form.type === "ip_allow") && (
              <div>
                <label className="block text-xs text-neutral-400 mb-1">IP Addresses / CIDRs (one per line)</label>
                <textarea
                  value={(form.config.ip_addresses ?? []).join("\n")}
                  onChange={(e) =>
                    updateConfig("ip_addresses", e.target.value.split("\n").filter(Boolean))
                  }
                  rows={4}
                  placeholder="10.0.0.0/8&#10;192.168.1.0/24"
                  className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                />
              </div>
            )}
            {form.type === "body_limit" && (
              <div>
                <label className="block text-xs text-neutral-400 mb-1">Max Body Size (KB)</label>
                <input
                  type="number"
                  value={form.config.max_body_size_kb ?? 1024}
                  onChange={(e) => updateConfig("max_body_size_kb", parseInt(e.target.value) || 0)}
                  className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                />
              </div>
            )}
            {form.type === "geo_block" && (
              <div>
                <label className="block text-xs text-neutral-400 mb-1">Country Codes (comma separated, ISO 3166-1 alpha-2)</label>
                <input
                  type="text"
                  value={(form.config.countries ?? []).join(", ")}
                  onChange={(e) =>
                    updateConfig("countries", e.target.value.split(",").map((s) => s.trim()).filter(Boolean))
                  }
                  placeholder="CN, RU, KP"
                  className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                />
              </div>
            )}
            {form.type === "header_rule" && (
              <div className="space-y-3">
                <div>
                  <label className="block text-xs text-neutral-400 mb-1">Header Name</label>
                  <input
                    type="text"
                    value={form.config.header_name ?? ""}
                    onChange={(e) => updateConfig("header_name", e.target.value)}
                    placeholder="User-Agent"
                    className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-neutral-400 mb-1">Match Pattern (regex)</label>
                  <input
                    type="text"
                    value={form.config.header_match ?? ""}
                    onChange={(e) => updateConfig("header_match", e.target.value)}
                    placeholder="^$"
                    className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white font-mono placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-neutral-400 mb-1">Action</label>
                  <select
                    value={form.config.action ?? "block"}
                    onChange={(e) => updateConfig("action", e.target.value)}
                    className="w-full rounded-lg border border-border bg-surface-100 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                  >
                    <option value="block">Block</option>
                    <option value="allow">Allow</option>
                  </select>
                </div>
              </div>
            )}
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
