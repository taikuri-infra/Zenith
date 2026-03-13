"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { StatCard } from "@/components/stat-card";
import { Modal } from "@/components/modal";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { useToast } from "@/components/toast";
import { getApi } from "@/lib/get-api";
import { type ApiGateway, type GatewayRouteInfo, type GatewayGroup, type GatewayRoutePlugin, type GatewayCustomDomain, type GatewayAnalyticsOverview, type GatewayTimeSeriesResponse, type DeployApp, type AuthPool, type CreateRouteInput, type UpdateRouteInput } from "@/lib/api";
import { useState, useCallback } from "react";
import { Pencil, Trash2, AlertTriangle, Power, Globe, BarChart3, Plus } from "lucide-react";

/* ── method badge colors ── */
const methodColors: Record<string, string> = {
  GET: "bg-emerald-500/10 text-emerald-400",
  POST: "bg-blue-500/10 text-blue-400",
  PUT: "bg-amber-500/10 text-amber-400",
  DELETE: "bg-red-500/10 text-red-400",
};

/* ── plugin definitions (form-field-driven, no raw JSON) ── */
type FieldDef = { key: string; label: string; type: "text" | "number" | "toggle" | "lines"; placeholder?: string };

const PLUGIN_DEFS: Record<string, { description: string; fields: FieldDef[]; defaults: Record<string, unknown> }> = {
  cors: {
    description: "Cross-Origin Resource Sharing",
    fields: [
      { key: "allow_origins", label: "Allow Origins", type: "text", placeholder: "* or comma-separated" },
      { key: "allow_methods", label: "Allow Methods", type: "text", placeholder: "GET,POST,PUT,DELETE,OPTIONS" },
      { key: "allow_headers", label: "Allow Headers", type: "text", placeholder: "*" },
      { key: "max_age", label: "Max Age (sec)", type: "number" },
    ],
    defaults: { allow_origins: "*", allow_methods: "GET,POST,PUT,DELETE,OPTIONS", allow_headers: "*", max_age: 3600 },
  },
  "limit-count": {
    description: "Rate limiting by request count",
    fields: [
      { key: "count", label: "Max Requests", type: "number" },
      { key: "time_window", label: "Time Window (sec)", type: "number" },
      { key: "rejected_code", label: "Reject Status Code", type: "number" },
    ],
    defaults: { count: 100, time_window: 60, rejected_code: 429 },
  },
  "ip-restriction": {
    description: "Allow/deny by IP address",
    fields: [
      { key: "whitelist", label: "Allowed IPs (one per line)", type: "lines", placeholder: "10.0.0.0/8\n192.168.1.0/24" },
    ],
    defaults: { whitelist: "" },
  },
  "proxy-rewrite": {
    description: "Rewrite upstream URI",
    fields: [
      { key: "regex_from", label: "Match Pattern", type: "text", placeholder: "^/api/v1(.*)" },
      { key: "regex_to", label: "Replace With", type: "text", placeholder: "/v1$1" },
    ],
    defaults: { regex_from: "", regex_to: "" },
  },
  "request-id": {
    description: "Add unique request ID header",
    fields: [
      { key: "header_name", label: "Header Name", type: "text", placeholder: "X-Request-Id" },
      { key: "include_in_response", label: "Include in Response", type: "toggle" },
    ],
    defaults: { header_name: "X-Request-Id", include_in_response: true },
  },
  "key-auth": { description: "API key authentication", fields: [], defaults: {} },
  "jwt-auth": { description: "JWT authentication", fields: [], defaults: {} },
};

/* ── serialize form config → APISIX plugin config ── */
function serializePluginConfig(name: string, config: Record<string, unknown>): Record<string, unknown> {
  if (name === "ip-restriction") {
    return { whitelist: String(config.whitelist ?? "").split("\n").map(s => s.trim()).filter(Boolean) };
  }
  if (name === "proxy-rewrite") {
    const from = String(config.regex_from ?? "");
    const to = String(config.regex_to ?? "");
    return from ? { regex_uri: [from, to] } : {};
  }
  return { ...config };
}

type PluginEntry = { name: string; enable: boolean; config: Record<string, unknown> };

function serializePlugins(plugins: PluginEntry[]): GatewayRoutePlugin[] {
  return plugins.filter(p => p.enable).map(p => ({
    name: p.name,
    enable: true,
    config: serializePluginConfig(p.name, p.config),
  }));
}

/* ── deserialize APISIX plugin config → form fields ── */
function deserializePluginConfig(name: string, config: Record<string, unknown>): Record<string, unknown> {
  if (name === "ip-restriction") {
    const wl = config.whitelist;
    return { whitelist: Array.isArray(wl) ? wl.join("\n") : String(wl ?? "") };
  }
  if (name === "proxy-rewrite") {
    const uri = config.regex_uri;
    if (Array.isArray(uri) && uri.length >= 2) {
      return { regex_from: String(uri[0]), regex_to: String(uri[1]) };
    }
    return { regex_from: "", regex_to: "" };
  }
  return { ...config };
}

function deserializePlugins(plugins: GatewayRoutePlugin[]): PluginEntry[] {
  return plugins.map(p => ({
    name: p.name,
    enable: p.enable !== false,
    config: deserializePluginConfig(p.name, p.config ?? {}),
  }));
}

/* ── plugin editor component (form fields, NOT raw JSON) ── */
function PluginEditor({ plugins, onChange }: { plugins: PluginEntry[]; onChange: (p: PluginEntry[]) => void }) {
  const usedNames = new Set(plugins.map(p => p.name));
  const available = Object.entries(PLUGIN_DEFS).filter(([n]) => !usedNames.has(n));
  const [showPicker, setShowPicker] = useState(false);

  const addPlugin = (name: string) => {
    onChange([...plugins, { name, enable: true, config: { ...PLUGIN_DEFS[name].defaults } }]);
    setShowPicker(false);
  };

  const update = (idx: number, key: string, value: unknown) =>
    onChange(plugins.map((p, i) => i === idx ? { ...p, config: { ...p.config, [key]: value } } : p));

  return (
    <div className="space-y-2">
      <label className="text-xs font-medium text-neutral-400">Plugins</label>

      {/* Existing plugins */}
      {plugins.map((plugin, idx) => {
        const def = PLUGIN_DEFS[plugin.name];
        return (
          <div key={plugin.name} className="rounded-md border border-border bg-surface-200 p-3 space-y-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => onChange(plugins.map((p, i) => i === idx ? { ...p, enable: !p.enable } : p))}
                  className={`h-4 w-7 rounded-full transition-colors ${plugin.enable ? "bg-accent-500" : "bg-neutral-600"}`}
                >
                  <span className={`block h-3 w-3 rounded-full bg-white transition-transform ${plugin.enable ? "translate-x-3.5" : "translate-x-0.5"}`} />
                </button>
                <span className="text-xs font-medium text-white">{plugin.name}</span>
                <span className="text-[10px] text-neutral-500">{def?.description}</span>
              </div>
              <button type="button" onClick={() => onChange(plugins.filter((_, i) => i !== idx))} className="text-xs text-red-400 hover:text-red-300">Remove</button>
            </div>
            {def?.fields.length ? (
              <div className="grid grid-cols-2 gap-2">
                {def.fields.map(f => (
                  <div key={f.key} className={f.type === "lines" ? "col-span-2" : ""}>
                    {f.type === "toggle" ? (
                      <label className="flex items-center gap-2 text-xs text-neutral-300 pt-1">
                        <input type="checkbox" checked={Boolean(plugin.config[f.key])} onChange={e => update(idx, f.key, e.target.checked)} className="rounded border-border bg-surface-300" />
                        {f.label}
                      </label>
                    ) : f.type === "lines" ? (
                      <>
                        <label className="mb-0.5 block text-[10px] text-neutral-500">{f.label}</label>
                        <textarea value={String(plugin.config[f.key] ?? "")} onChange={e => update(idx, f.key, e.target.value)} rows={3} placeholder={f.placeholder}
                          className="w-full rounded-md border border-border bg-surface-300 px-2 py-1.5 text-xs text-neutral-300 placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                      </>
                    ) : (
                      <>
                        <label className="mb-0.5 block text-[10px] text-neutral-500">{f.label}</label>
                        <input type={f.type === "number" ? "number" : "text"} value={plugin.config[f.key] as string | number ?? ""}
                          onChange={e => update(idx, f.key, f.type === "number" ? Number(e.target.value) : e.target.value)}
                          placeholder={f.placeholder}
                          className="w-full rounded-md border border-border bg-surface-300 px-2 py-1.5 text-xs text-neutral-300 placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
                      </>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-[10px] text-neutral-600">No configuration needed</p>
            )}
          </div>
        );
      })}

      {/* Add plugin button + picker */}
      {available.length > 0 && (
        showPicker ? (
          <div className="rounded-md border border-accent-500/30 bg-surface-200 p-2 space-y-1">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[10px] font-medium text-neutral-400">Select a plugin to add</span>
              <button type="button" onClick={() => setShowPicker(false)} className="text-[10px] text-neutral-500 hover:text-neutral-300">Cancel</button>
            </div>
            {available.map(([name, def]) => (
              <button
                key={name}
                type="button"
                onClick={() => addPlugin(name)}
                className="flex w-full items-center justify-between rounded-md px-2.5 py-1.5 text-left hover:bg-surface-300 transition-colors"
              >
                <span className="text-xs font-medium text-white">{name}</span>
                <span className="text-[10px] text-neutral-500">{def.description}</span>
              </button>
            ))}
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setShowPicker(true)}
            className="flex w-full items-center justify-center gap-1.5 rounded-md border border-dashed border-border py-2 text-xs text-neutral-400 hover:border-accent-500/50 hover:text-accent-400 transition-colors"
          >
            <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            Add Plugin
          </button>
        )
      )}

      {plugins.length === 0 && !showPicker && available.length === 0 && (
        <p className="text-xs text-neutral-600">No plugins available</p>
      )}
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════ */
/*  Main Page                                                    */
/* ══════════════════════════════════════════════════════════════ */

export default function GatewayPage() {
  const { gateways, appsDeploy, authPools } = getApi();
  const projectId = useProject();
  const { toast } = useToast();

  const { data: gwList, loading: gwLoading, error: gwError, refetch: gwRefetch } = useApi(() => gateways.list(projectId || undefined), [projectId]);
  const { data: appsData } = useApi(() => appsDeploy.list(projectId || undefined), [projectId]);
  const userApps: DeployApp[] = appsData?.items ?? [];
  const { data: poolsData } = useApi(() => authPools.list(), []);
  const pools: AuthPool[] = (poolsData ?? []).filter((p: AuthPool) => p.status === "active");

  const [selectedGwId, setSelectedGwId] = useState<string | null>(null);
  const activeGw = gwList?.find((g: ApiGateway) => g.id === selectedGwId) ?? gwList?.[0] ?? null;

  const { data: routes, loading: routesLoading, refetch: routesRefetch } = useApi(
    () => activeGw ? gateways.listRoutes(activeGw.id) : Promise.resolve([]), [activeGw?.id]);
  const { data: groups, refetch: groupsRefetch } = useApi(
    () => activeGw ? gateways.listGroups(activeGw.id) : Promise.resolve([]), [activeGw?.id]);

  const routeList: GatewayRouteInfo[] = routes ?? [];
  const groupList: GatewayGroup[] = groups ?? [];

  // Custom Domains
  const { data: domains, refetch: domainsRefetch } = useApi(
    () => activeGw ? gateways.listDomains(activeGw.id) : Promise.resolve([]), [activeGw?.id]);
  const domainList: GatewayCustomDomain[] = domains ?? [];

  // Analytics
  const { data: analyticsData } = useApi(
    () => activeGw ? gateways.getAnalytics(activeGw.id) : Promise.resolve(null), [activeGw?.id]);
  const analytics: GatewayAnalyticsOverview | null = analyticsData ?? null;

  const [analyticsRange, setAnalyticsRange] = useState("1h");
  const { data: timeSeriesData } = useApi(
    () => activeGw ? gateways.getAnalyticsTimeSeries(activeGw.id, "requests", analyticsRange) : Promise.resolve(null), [activeGw?.id, analyticsRange]);
  const timeSeries: GatewayTimeSeriesResponse | null = timeSeriesData ?? null;

  /* expanded group cards */
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const toggleExpand = (id: string) => setExpandedGroups(prev => {
    const next = new Set(prev);
    next.has(id) ? next.delete(id) : next.add(id);
    return next;
  });

  /* ── Create Gateway ── */
  const [showCreateGw, setShowCreateGw] = useState(false);
  const [newGwName, setNewGwName] = useState("");
  const [creating, setCreating] = useState(false);
  const handleCreateGw = useCallback(async () => {
    if (!newGwName.trim()) return;
    setCreating(true);
    try {
      const gw = await gateways.create(newGwName.trim());
      setShowCreateGw(false); setNewGwName(""); setSelectedGwId(gw.id); gwRefetch();
      toast("success", "Gateway created");
    } catch {
      toast("error", "Failed to create gateway");
    } finally { setCreating(false); }
  }, [newGwName, gateways, gwRefetch, toast]);

  /* ── Delete Gateway (confirmation dialog) ── */
  const [showDeleteGw, setShowDeleteGw] = useState(false);
  const [deleteGwConfirm, setDeleteGwConfirm] = useState("");
  const [deletingGw, setDeletingGw] = useState(false);

  const handleDeleteGw = useCallback(async () => {
    if (!activeGw) return;
    setDeletingGw(true);
    try {
      await gateways.delete(activeGw.id);
      setShowDeleteGw(false); setDeleteGwConfirm(""); setSelectedGwId(null); gwRefetch();
      toast("success", "Gateway deleted");
    } catch {
      toast("error", "Failed to delete gateway");
    } finally { setDeletingGw(false); }
  }, [activeGw, gateways, gwRefetch, toast]);

  /* ── Sync Gateway ── */
  const handleSyncGw = useCallback(async () => {
    if (!activeGw) return;
    try {
      await gateways.sync(activeGw.id);
      routesRefetch(); groupsRefetch();
      toast("success", "Gateway synced");
    } catch {
      toast("error", "Failed to sync");
    }
  }, [activeGw, gateways, routesRefetch, groupsRefetch, toast]);

  /* ── Custom Domains ── */
  const [showAddDomain, setShowAddDomain] = useState(false);
  const [newDomain, setNewDomain] = useState("");
  const [addingDomain, setAddingDomain] = useState(false);

  const handleAddDomain = useCallback(async () => {
    if (!activeGw || !newDomain.trim()) return;
    setAddingDomain(true);
    try {
      await gateways.addDomain(activeGw.id, newDomain.trim());
      setShowAddDomain(false); setNewDomain("");
      domainsRefetch();
      toast("success", "Domain added");
    } catch {
      toast("error", "Failed to add domain");
    } finally { setAddingDomain(false); }
  }, [activeGw, newDomain, gateways, domainsRefetch, toast]);

  const handleDeleteDomain = useCallback(async (domainId: string) => {
    if (!activeGw) return;
    try {
      await gateways.deleteDomain(activeGw.id, domainId);
      domainsRefetch();
      toast("success", "Domain deleted");
    } catch {
      toast("error", "Failed to delete domain");
    }
  }, [activeGw, gateways, domainsRefetch, toast]);

  /* ── Add Route (pre-selects group when opened from group card) ── */
  const [showAddRoute, setShowAddRoute] = useState(false);
  const [routeName, setRouteName] = useState("");
  const [routePath, setRoutePath] = useState("");
  const [routeAppId, setRouteAppId] = useState("");
  const [routeGroupId, setRouteGroupId] = useState("");
  const [routeMethods, setRouteMethods] = useState<Record<string, boolean>>({ GET: true });
  const [routeStripPrefix, setRouteStripPrefix] = useState(false);
  const [routeAuthPoolId, setRouteAuthPoolId] = useState("");
  const [routePlugins, setRoutePlugins] = useState<PluginEntry[]>([]);
  const [addingRoute, setAddingRoute] = useState(false);

  const openAddRoute = (preGroupId?: string) => {
    setRouteGroupId(preGroupId ?? "");
    setRouteAppId("");
    setRouteName(""); setRoutePath("");
    setRouteMethods({ GET: true }); setRouteStripPrefix(false);
    setRouteAuthPoolId(""); setRoutePlugins([]);
    setShowAddRoute(true);
  };

  const handleAddRoute = useCallback(async () => {
    if (!activeGw || !routeName.trim() || !routePath.trim()) return;
    if (!routeAppId && !routeGroupId) return;
    const methods = Object.entries(routeMethods).filter(([, v]) => v).map(([k]) => k);
    if (methods.length === 0) methods.push("GET");
    setAddingRoute(true);
    try {
      const data: CreateRouteInput = {
        name: routeName.trim(),
        path: routePath.trim(),
        methods,
        strip_prefix: routeStripPrefix,
      };
      if (routeGroupId) data.group_id = routeGroupId;
      else data.app_id = routeAppId;
      if (routeAuthPoolId) data.auth_pool_id = routeAuthPoolId;
      const serialized = serializePlugins(routePlugins);
      if (serialized.length > 0) data.plugins = serialized;
      await gateways.createRoute(activeGw.id, data);
      setShowAddRoute(false);
      routesRefetch(); gwRefetch();
      toast("success", "Route created");
    } catch {
      toast("error", "Failed to create route");
    } finally { setAddingRoute(false); }
  }, [activeGw, routeName, routePath, routeAppId, routeGroupId, routeMethods, routeStripPrefix, routeAuthPoolId, routePlugins, gateways, routesRefetch, gwRefetch, toast]);

  /* ── Delete Route (confirmation dialog) ── */
  const [deleteRouteTarget, setDeleteRouteTarget] = useState<GatewayRouteInfo | null>(null);
  const [deletingRoute, setDeletingRoute] = useState(false);

  const handleDeleteRoute = useCallback(async (routeId: string) => {
    if (!activeGw) return;
    setDeletingRoute(true);
    try {
      await gateways.deleteRoute(activeGw.id, routeId);
      setDeleteRouteTarget(null);
      routesRefetch(); gwRefetch();
      toast("success", "Route deleted");
    } catch {
      toast("error", "Failed to delete route");
    } finally { setDeletingRoute(false); }
  }, [activeGw, gateways, routesRefetch, gwRefetch, toast]);

  /* ── Add Group ── */
  const [showAddGroup, setShowAddGroup] = useState(false);
  const [groupName, setGroupName] = useState("");
  const [groupAppId, setGroupAppId] = useState("");
  const [groupPlugins, setGroupPlugins] = useState<PluginEntry[]>([]);
  const [addingGroup, setAddingGroup] = useState(false);

  const handleAddGroup = useCallback(async () => {
    if (!activeGw || !groupName.trim() || !groupAppId) return;
    setAddingGroup(true);
    try {
      const serialized = serializePlugins(groupPlugins);
      await gateways.createGroup(activeGw.id, {
        name: groupName.trim(),
        app_id: groupAppId,
        ...(serialized.length > 0 ? { plugins: serialized } : {}),
      });
      setShowAddGroup(false); setGroupName(""); setGroupAppId(""); setGroupPlugins([]);
      groupsRefetch();
      toast("success", "Group created");
    } catch {
      toast("error", "Failed to create group");
    } finally { setAddingGroup(false); }
  }, [activeGw, groupName, groupAppId, groupPlugins, gateways, groupsRefetch, toast]);

  /* ── Delete Group (confirmation dialog) ── */
  const [deleteGroupTarget, setDeleteGroupTarget] = useState<GatewayGroup | null>(null);
  const [deleteGroupConfirm, setDeleteGroupConfirm] = useState("");
  const [deletingGroup, setDeletingGroup] = useState(false);

  const handleDeleteGroup = useCallback(async (groupId: string) => {
    if (!activeGw) return;
    setDeletingGroup(true);
    try {
      await gateways.deleteGroup(activeGw.id, groupId);
      setDeleteGroupTarget(null); setDeleteGroupConfirm("");
      groupsRefetch(); routesRefetch();
      toast("success", "Group deleted");
    } catch {
      toast("error", "Failed to delete group");
    } finally { setDeletingGroup(false); }
  }, [activeGw, gateways, groupsRefetch, routesRefetch, toast]);

  /* ── Edit Group ── */
  const [editGroup, setEditGroup] = useState<GatewayGroup | null>(null);
  const [editGroupName, setEditGroupName] = useState("");
  const [editGroupAppId, setEditGroupAppId] = useState("");
  const [editGroupPlugins, setEditGroupPlugins] = useState<PluginEntry[]>([]);
  const [savingGroup, setSavingGroup] = useState(false);

  const openEditGroup = (group: GatewayGroup) => {
    setEditGroup(group);
    setEditGroupName(group.name);
    setEditGroupAppId(group.app_id);
    setEditGroupPlugins(deserializePlugins(group.plugins));
  };

  const handleUpdateGroup = useCallback(async () => {
    if (!activeGw || !editGroup || !editGroupName.trim() || !editGroupAppId) return;
    setSavingGroup(true);
    try {
      const serialized = serializePlugins(editGroupPlugins);
      await gateways.updateGroup(activeGw.id, editGroup.id, {
        name: editGroupName.trim(),
        app_id: editGroupAppId,
        plugins: serialized,
      });
      setEditGroup(null);
      groupsRefetch(); routesRefetch();
      toast("success", "Group updated");
    } catch {
      toast("error", "Failed to update group");
    } finally { setSavingGroup(false); }
  }, [activeGw, editGroup, editGroupName, editGroupAppId, editGroupPlugins, gateways, groupsRefetch, routesRefetch, toast]);

  /* ── Edit Route ── */
  const [editRoute, setEditRoute] = useState<GatewayRouteInfo | null>(null);
  const [editRouteName, setEditRouteName] = useState("");
  const [editRoutePath, setEditRoutePath] = useState("");
  const [editRouteAppId, setEditRouteAppId] = useState("");
  const [editRouteGroupId, setEditRouteGroupId] = useState("");
  const [editRouteMethods, setEditRouteMethods] = useState<Record<string, boolean>>({});
  const [editRouteStripPrefix, setEditRouteStripPrefix] = useState(false);
  const [editRouteAuthPoolId, setEditRouteAuthPoolId] = useState("");
  const [editRoutePlugins, setEditRoutePlugins] = useState<PluginEntry[]>([]);
  const [savingRoute, setSavingRoute] = useState(false);

  const openEditRoute = (route: GatewayRouteInfo) => {
    setEditRoute(route);
    setEditRouteName(route.name);
    setEditRoutePath(route.path);
    setEditRouteAppId(route.app_id ?? "");
    setEditRouteGroupId(route.group_id ?? "");
    const methods: Record<string, boolean> = {};
    route.methods.forEach(m => { methods[m] = true; });
    setEditRouteMethods(methods);
    setEditRouteStripPrefix(route.strip_prefix ?? false);
    setEditRouteAuthPoolId(route.auth_pool_id ?? "");
    setEditRoutePlugins(deserializePlugins(route.plugins));
  };

  const handleUpdateRoute = useCallback(async () => {
    if (!activeGw || !editRoute || !editRouteName.trim() || !editRoutePath.trim()) return;
    const methods = Object.entries(editRouteMethods).filter(([, v]) => v).map(([k]) => k);
    if (methods.length === 0) methods.push("GET");
    setSavingRoute(true);
    try {
      const data: UpdateRouteInput = {
        name: editRouteName.trim(),
        path: editRoutePath.trim(),
        methods,
        strip_prefix: editRouteStripPrefix,
        plugins: serializePlugins(editRoutePlugins),
      };
      if (editRouteGroupId) {
        data.group_id = editRouteGroupId;
      } else {
        data.app_id = editRouteAppId;
      }
      if (editRouteAuthPoolId) {
        data.auth = "oidc";
        data.auth_pool_id = editRouteAuthPoolId;
      } else {
        data.auth = "";
        data.auth_pool_id = "";
      }
      await gateways.updateRoute(activeGw.id, editRoute.id, data);
      setEditRoute(null);
      routesRefetch(); gwRefetch();
      toast("success", "Route updated");
    } catch {
      toast("error", "Failed to update route");
    } finally { setSavingRoute(false); }
  }, [activeGw, editRoute, editRouteName, editRoutePath, editRouteAppId, editRouteGroupId, editRouteMethods, editRouteStripPrefix, editRouteAuthPoolId, editRoutePlugins, gateways, routesRefetch, gwRefetch, toast]);

  /* ── Toggle Route Status ── */
  const handleToggleRouteStatus = useCallback(async (route: GatewayRouteInfo) => {
    if (!activeGw) return;
    const newStatus = route.status === "active" ? "stopped" : "active";
    try {
      await gateways.updateRoute(activeGw.id, route.id, { status: newStatus });
      routesRefetch();
      toast("success", newStatus === "active" ? "Route activated" : "Route stopped");
    } catch {
      toast("error", "Failed to update status");
    }
  }, [activeGw, gateways, routesRefetch, toast]);

  /* ── Form validation helpers ── */
  const routeNameHasSpaces = (name: string) => name.includes(" ");
  const routePathInvalid = (path: string) => path.length > 0 && !path.startsWith("/");
  const noMethodsSelected = (methods: Record<string, boolean>) => !Object.values(methods).some(Boolean);

  // Add route validation
  const addRouteValid = routeName.trim().length > 0
    && routePath.trim().length > 0
    && !routeNameHasSpaces(routeName)
    && !routePathInvalid(routePath)
    && !noMethodsSelected(routeMethods)
    && (!!routeAppId || !!routeGroupId);

  // Edit route validation
  const editRouteValid = editRouteName.trim().length > 0
    && editRoutePath.trim().length > 0
    && !routeNameHasSpaces(editRouteName)
    && !routePathInvalid(editRoutePath)
    && !noMethodsSelected(editRouteMethods)
    && (!!editRouteAppId || !!editRouteGroupId);

  /* ── Render ── */

  if (gwLoading) return <Shell><PageWithTableSkeleton cols={6} rows={4} /></Shell>;
  if (gwError) return <Shell><ErrorState message={gwError} onRetry={gwRefetch} /></Shell>;

  const gatewayList: ApiGateway[] = gwList ?? [];
  const standaloneRoutes = routeList.filter(r => !r.group_id);

  const routeRow = (route: GatewayRouteInfo, groupPlugins?: GatewayRoutePlugin[]) => {
    const routePluginNames = new Set(route.plugins.map(p => p.name));
    return (
      <tr key={route.id} className="border-b border-border last:border-0 hover:bg-surface-200/50 transition-colors">
        <td className="px-4 py-2.5 text-sm font-medium text-white">{route.name}</td>
        <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">{route.path}</td>
        <td className="px-4 py-2.5">
          <div className="flex flex-wrap gap-1">
            {route.methods.map(m => (
              <span key={m} className={`inline-flex rounded px-1.5 py-0.5 text-[10px] font-semibold ${methodColors[m] || "bg-neutral-500/10 text-neutral-400"}`}>{m}</span>
            ))}
          </div>
        </td>
        <td className="px-4 py-2.5">
          <div className="flex flex-wrap gap-1">
            {/* Inherited group plugins (faded accent for non-overridden) */}
            {groupPlugins?.map((p, i) => (
              <span key={`g-${p.name}-${i}`} className={`inline-flex rounded px-1.5 py-0.5 text-[10px] ${routePluginNames.has(p.name) ? "line-through opacity-40 bg-accent-500/10 text-accent-400" : "bg-accent-500/10 text-accent-400 opacity-60"}`}>
                {p.name}
              </span>
            ))}
            {/* Route-level plugins */}
            {route.plugins.map((p, i) => (
              <span key={`r-${p.name}-${i}`} className="inline-flex rounded bg-blue-500/10 px-1.5 py-0.5 text-[10px] text-blue-400">{p.name}</span>
            ))}
            {route.auth === "oidc" && <span className="inline-flex rounded bg-purple-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-purple-400">OIDC</span>}
            {route.strip_prefix && <span className="inline-flex rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] text-amber-400">strip-prefix</span>}
          </div>
        </td>
        <td className="px-4 py-2.5"><StatusBadge status={route.status === "active" ? "running" : "stopped"} /></td>
        <td className="px-4 py-2.5">
          <div className="flex items-center gap-1.5">
            <button onClick={() => handleToggleRouteStatus(route)} className={`p-1 rounded transition-colors ${route.status === "active" ? "text-emerald-400 hover:text-emerald-300" : "text-neutral-500 hover:text-neutral-300"}`} title={route.status === "active" ? "Stop route" : "Activate route"}>
              <Power className="h-3.5 w-3.5" />
            </button>
            <button onClick={() => openEditRoute(route)} className="p-1 text-neutral-400 hover:text-white rounded transition-colors" title="Edit route">
              <Pencil className="h-3.5 w-3.5" />
            </button>
            <button onClick={() => setDeleteRouteTarget(route)} className="p-1 text-red-400 hover:text-red-300 rounded transition-colors" title="Delete route">
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        </td>
      </tr>
    );
  };

  const routeTableHead = (
    <thead>
      <tr className="border-b border-border bg-surface-100/50">
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Name</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Path</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Methods</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Plugins</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Status</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Actions</th>
      </tr>
    </thead>
  );

  /* ── Inline validation hint ── */
  const ValidationHint = ({ show, color = "red", children }: { show: boolean; color?: "red" | "amber"; children: React.ReactNode }) =>
    show ? <p className={`mt-0.5 text-[10px] ${color === "red" ? "text-red-400" : "text-amber-400"}`}>{children}</p> : null;

  /* ── Route form fields (shared between Add and Edit) ── */
  const RouteFormFields = ({
    name, setName, path, setPath, methods, setMethods, appId, setAppId, gId, setGId,
    stripPrefix, setStripPrefix, authPoolId, setAuthPoolId, plugins, setPlugins,
    isEdit,
  }: {
    name: string; setName: (v: string) => void;
    path: string; setPath: (v: string) => void;
    methods: Record<string, boolean>; setMethods: (v: Record<string, boolean>) => void;
    appId: string; setAppId: (v: string) => void;
    gId: string; setGId: (v: string) => void;
    stripPrefix: boolean; setStripPrefix: (v: boolean) => void;
    authPoolId: string; setAuthPoolId: (v: string) => void;
    plugins: PluginEntry[]; setPlugins: (v: PluginEntry[]) => void;
    isEdit: boolean;
  }) => {
    const inGroup = !!gId;
    const group = groupList.find(g => g.id === gId);
    return (
      <>
        <div>
          <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
          <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="users-api"
            className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" required autoFocus />
          <ValidationHint show={routeNameHasSpaces(name)}>Route name should not contain spaces</ValidationHint>
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-neutral-400">Path</label>
          <input type="text" value={path} onChange={e => setPath(e.target.value)} placeholder="/api/v1/users/*"
            className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" required />
          <ValidationHint show={routePathInvalid(path)}>Path must start with /</ValidationHint>
        </div>

        {/* Group selector — always visible in edit mode to allow moving */}
        {(isEdit || (!gId && groupList.length > 0)) && (
          <div>
            <label className="mb-1 block text-xs font-medium text-neutral-400">Group {isEdit ? "" : "(optional)"}</label>
            <select value={gId} onChange={e => { setGId(e.target.value); if (e.target.value) setAppId(""); }}
              className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none">
              <option value="">Standalone (select target app below)</option>
              {groupList.map(g => <option key={g.id} value={g.id}>{g.name} ({g.app_subdomain})</option>)}
            </select>
          </div>
        )}

        {/* Target app — only when standalone */}
        {!gId && (
          <div>
            <label className="mb-1 block text-xs font-medium text-neutral-400">Target App</label>
            <select value={appId} onChange={e => setAppId(e.target.value)}
              className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none" required={!gId}>
              <option value="">Select an app...</option>
              {userApps.map(app => <option key={app.id} value={app.id}>{app.name} ({app.subdomain})</option>)}
            </select>
          </div>
        )}

        {inGroup && (
          <div className="rounded-md bg-accent-500/5 border border-accent-500/20 px-3 py-2">
            <p className="text-xs text-accent-400">Target app inherited from group{group ? ` (${group.app_subdomain})` : ""}. Route-level plugins override group plugins with the same name.</p>
          </div>
        )}

        {/* Effective plugins hint (edit mode, grouped routes) */}
        {isEdit && inGroup && group && group.plugins.length > 0 && (
          <div className="rounded-md bg-surface-200 border border-border px-3 py-2 space-y-1">
            <p className="text-[10px] font-medium text-neutral-500 uppercase tracking-wider">Effective Plugins</p>
            <div className="flex flex-wrap gap-1">
              {group.plugins.map((p, i) => {
                const overridden = plugins.some(rp => rp.name === p.name);
                return (
                  <span key={`eg-${p.name}-${i}`} className={`inline-flex rounded px-1.5 py-0.5 text-[10px] ${overridden ? "line-through opacity-40 bg-accent-500/10 text-accent-400" : "bg-accent-500/10 text-accent-400"}`}>
                    {p.name} (group)
                  </span>
                );
              })}
              {plugins.filter(p => p.enable).map((p, i) => (
                <span key={`er-${p.name}-${i}`} className="inline-flex rounded bg-blue-500/10 px-1.5 py-0.5 text-[10px] text-blue-400">
                  {p.name} (route)
                </span>
              ))}
            </div>
          </div>
        )}

        <div>
          <label className="mb-1 block text-xs font-medium text-neutral-400">Methods</label>
          <div className="flex gap-4 pt-1">
            {["GET", "POST", "PUT", "DELETE"].map(m => (
              <label key={m} className="flex items-center gap-1.5 text-xs text-neutral-300">
                <input type="checkbox" checked={methods[m] || false} onChange={e => setMethods({ ...methods, [m]: e.target.checked })} className="rounded border-border bg-surface-200" />
                {m}
              </label>
            ))}
          </div>
          <ValidationHint show={noMethodsSelected(methods)} color="amber">Select at least one method</ValidationHint>
        </div>
        <label className="flex items-center gap-2 text-xs text-neutral-300">
          <input type="checkbox" checked={stripPrefix} onChange={e => setStripPrefix(e.target.checked)} className="rounded border-border bg-surface-200" />
          Strip path prefix before forwarding
        </label>
        {pools.length > 0 && (
          <div>
            <label className="mb-1 block text-xs font-medium text-neutral-400">Auth Pool (OIDC)</label>
            <select value={authPoolId} onChange={e => setAuthPoolId(e.target.value)}
              className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none">
              <option value="">None</option>
              {pools.map(pool => <option key={pool.id} value={pool.id}>{pool.name} ({pool.user_count} users)</option>)}
            </select>
          </div>
        )}
        <PluginEditor plugins={plugins} onChange={setPlugins} />
      </>
    );
  };

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">API Gateway</h1>
            <p className="text-sm text-neutral-500">APISIX-powered traffic management and routing</p>
          </div>
          <button onClick={() => setShowCreateGw(true)} className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
            + New Gateway
          </button>
        </div>

        {/* Gateway selector */}
        {gatewayList.length > 0 && (
          <div className="flex items-center gap-3">
            <label className="text-xs font-medium text-neutral-400">Gateway:</label>
            <select value={activeGw?.id ?? ""} onChange={e => setSelectedGwId(e.target.value)}
              className="rounded-md border border-border bg-surface-200 px-3 py-1.5 text-sm text-white focus:border-accent-500 focus:outline-none">
              {gatewayList.map(gw => <option key={gw.id} value={gw.id}>{gw.name} ({gw.slug})</option>)}
            </select>
            {activeGw && (
              <>
                <StatusBadge status={activeGw.status === "active" ? "running" : activeGw.status === "error" ? "error" : "pending"} />
                <span className="font-mono text-xs text-neutral-500">{activeGw.endpoint}</span>
                <button onClick={handleSyncGw} className="ml-auto rounded-md border border-border px-2.5 py-1 text-xs text-neutral-400 hover:text-white transition-colors" title="Force reconcile K8s CRDs">Sync</button>
                <button onClick={() => { setShowDeleteGw(true); setDeleteGwConfirm(""); }} className="rounded-md border border-red-500/30 px-2.5 py-1 text-xs text-red-400 hover:bg-red-500/10 transition-colors">Delete</button>
              </>
            )}
          </div>
        )}

        {gatewayList.length === 0 ? (
          <EmptyState title="No gateways yet" description="Create your first API gateway to route traffic to your apps with plugins." actionLabel="Create Gateway" onAction={() => setShowCreateGw(true)} />
        ) : activeGw ? (
          <>
            {/* Stats */}
            <div className="grid grid-cols-4 gap-4">
              <StatCard label="Endpoint" value={activeGw.slug + ".gw.*"} />
              <StatCard label="Status" value={activeGw.status} />
              <StatCard label="Routes" value={String(routeList.length)} />
              <StatCard label="Groups" value={String(groupList.length)} />
            </div>

            {/* Action buttons */}
            <div className="flex gap-3">
              <button onClick={() => { setShowAddGroup(true); setGroupName(""); setGroupAppId(""); setGroupPlugins([]); }}
                className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
                + New Group
              </button>
              <button onClick={() => openAddRoute()}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-neutral-300 hover:text-white hover:border-neutral-500 transition-colors">
                + Standalone Route
              </button>
            </div>

            {/* ── Groups with nested routes ── */}
            {routesLoading ? (
              <div className="rounded-lg border border-border p-8 text-center text-sm text-neutral-500">Loading...</div>
            ) : (
              <div className="space-y-4">
                {groupList.map(group => {
                  const groupRoutes = routeList.filter(r => r.group_id === group.id);
                  const isExpanded = expandedGroups.has(group.id);
                  return (
                    <div key={group.id} className="rounded-lg border border-border bg-surface-100 overflow-hidden">
                      {/* Group header */}
                      <button
                        type="button"
                        onClick={() => toggleExpand(group.id)}
                        className="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-surface-200/50 transition-colors"
                      >
                        <div className="flex items-center gap-3">
                          <svg className={`h-4 w-4 text-neutral-500 transition-transform ${isExpanded ? "rotate-90" : ""}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                          </svg>
                          <div>
                            <span className="text-sm font-medium text-white">{group.name}</span>
                            <span className="ml-2 font-mono text-xs text-neutral-500">{group.app_subdomain}</span>
                          </div>
                          <span className="rounded-full bg-surface-300 px-2 py-0.5 text-[10px] text-neutral-400">
                            {groupRoutes.length} route{groupRoutes.length !== 1 ? "s" : ""}
                          </span>
                        </div>
                        <div className="flex items-center gap-2">
                          {group.plugins.map((p, i) => (
                            <span key={`${p.name}-${i}`} className="inline-flex rounded bg-accent-500/10 px-1.5 py-0.5 text-[10px] font-medium text-accent-400">{p.name}</span>
                          ))}
                          <button
                            type="button"
                            onClick={e => { e.stopPropagation(); openEditGroup(group); }}
                            className="ml-2 p-1 text-neutral-400 hover:text-white rounded transition-colors"
                            title="Edit group"
                          ><Pencil className="h-3.5 w-3.5" /></button>
                          <button
                            type="button"
                            onClick={e => { e.stopPropagation(); setDeleteGroupTarget(group); setDeleteGroupConfirm(""); }}
                            className="p-1 text-red-400 hover:text-red-300 rounded transition-colors"
                            title="Delete group"
                          ><Trash2 className="h-3.5 w-3.5" /></button>
                        </div>
                      </button>

                      {/* Expanded: routes table + add route */}
                      {isExpanded && (
                        <div className="border-t border-border">
                          {groupRoutes.length > 0 ? (
                            <table className="w-full text-sm">
                              {routeTableHead}
                              <tbody>{groupRoutes.map(r => routeRow(r, group.plugins))}</tbody>
                            </table>
                          ) : (
                            <div className="px-4 py-6 text-center">
                              <p className="text-xs text-neutral-500">No routes in this group yet.</p>
                              <button onClick={() => openAddRoute(group.id)} className="mt-2 text-xs text-accent-400 hover:text-accent-300 transition-colors">
                                Add your first route to {group.name}
                              </button>
                            </div>
                          )}
                          {groupRoutes.length > 0 && (
                            <div className="border-t border-border px-4 py-2">
                              <button onClick={() => openAddRoute(group.id)} className="text-xs text-accent-400 hover:text-accent-300 transition-colors">
                                + Add Route to {group.name}
                              </button>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}

                {/* ── Standalone routes (no group) ── */}
                {standaloneRoutes.length > 0 && (
                  <div className="space-y-2">
                    <h3 className="text-xs font-medium uppercase tracking-wider text-neutral-500">Standalone Routes</h3>
                    <div className="overflow-hidden rounded-lg border border-border">
                      <table className="w-full text-sm">
                        {routeTableHead}
                        <tbody>{standaloneRoutes.map(r => routeRow(r))}</tbody>
                      </table>
                    </div>
                  </div>
                )}

                {/* ── Better empty state ── */}
                {groupList.length === 0 && standaloneRoutes.length === 0 && (
                  <div className="rounded-lg border border-border bg-surface-100 p-10 text-center">
                    <svg className="mx-auto h-10 w-10 text-neutral-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
                    </svg>
                    <h3 className="mt-3 text-sm font-medium text-white">No routes configured</h3>
                    <p className="mt-1 text-xs text-neutral-500">Create a group to bundle routes to one app with shared plugins, or add a standalone route for individual endpoints.</p>
                    <div className="mt-4 flex justify-center gap-3">
                      <button onClick={() => { setShowAddGroup(true); setGroupName(""); setGroupAppId(""); setGroupPlugins([]); }}
                        className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
                        + New Group
                      </button>
                      <button onClick={() => openAddRoute()}
                        className="rounded-lg border border-border px-3 py-1.5 text-sm text-neutral-300 hover:text-white hover:border-neutral-500 transition-colors">
                        + Standalone Route
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* ── Analytics Dashboard ── */}
            <section>
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-sm font-medium text-white flex items-center gap-1.5"><BarChart3 className="h-4 w-4 text-accent-400" /> Analytics</h2>
                <div className="flex gap-1">
                  {["1h", "6h", "24h", "7d"].map(r => (
                    <button key={r} onClick={() => setAnalyticsRange(r)}
                      className={`rounded px-2 py-0.5 text-[10px] font-medium transition-colors ${analyticsRange === r ? "bg-accent-500 text-white" : "bg-surface-200 text-neutral-400 hover:text-white"}`}>
                      {r}
                    </button>
                  ))}
                </div>
              </div>
              <div className="grid grid-cols-4 gap-3 mb-4">
                <StatCard label="Requests/min" value={analytics ? (analytics.request_rate * 60).toFixed(1) : "—"} />
                <StatCard label="Error Rate" value={analytics ? `${analytics.error_rate.toFixed(1)}%` : "—"} />
                <StatCard label="P95 Latency" value={analytics ? `${(analytics.p95_latency * 1000).toFixed(0)}ms` : "—"} />
                <StatCard label="Total 24h" value={analytics ? Math.round(analytics.total_requests_24h).toLocaleString() : "—"} />
              </div>
              {timeSeries && timeSeries.points.length > 0 && (
                <div className="rounded-lg border border-border bg-surface-100 p-4">
                  <p className="text-xs text-neutral-500 mb-2">Request Rate Over Time ({analyticsRange})</p>
                  <div className="flex items-end gap-px h-20">
                    {(() => {
                      const maxVal = Math.max(...timeSeries.points.map(p => p.value), 1);
                      return timeSeries.points.map((p, i) => (
                        <div key={i} className="flex-1 bg-accent-500/60 rounded-t-sm hover:bg-accent-500 transition-colors"
                          style={{ height: `${(p.value / maxVal) * 100}%`, minHeight: "2px" }}
                          title={`${p.value.toFixed(1)} req/min`} />
                      ));
                    })()}
                  </div>
                </div>
              )}
            </section>

            {/* ── Custom Domains ── */}
            <section>
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-sm font-medium text-white flex items-center gap-1.5"><Globe className="h-4 w-4 text-accent-400" /> Custom Domains</h2>
                <button onClick={() => { setNewDomain(""); setShowAddDomain(true); }}
                  className="flex items-center gap-1 rounded-lg bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 transition-colors">
                  <Plus className="h-3 w-3" /> Add Domain
                </button>
              </div>
              {domainList.length === 0 ? (
                <div className="rounded-lg border border-border p-6 text-center">
                  <p className="text-sm text-neutral-500">No custom domains</p>
                  <p className="mt-1 text-xs text-neutral-600">Add a custom domain to serve your gateway from your own hostname (Pro+ required).</p>
                </div>
              ) : (
                <div className="rounded-lg border border-border overflow-hidden">
                  <table className="w-full text-left">
                    <thead>
                      <tr className="border-b border-border bg-surface-100/50">
                        <th className="px-4 py-2 text-xs font-medium text-neutral-500">Domain</th>
                        <th className="px-4 py-2 text-xs font-medium text-neutral-500">Status</th>
                        <th className="px-4 py-2 text-xs font-medium text-neutral-500">TLS</th>
                        <th className="px-4 py-2 text-xs font-medium text-neutral-500">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {domainList.map(d => (
                        <tr key={d.id} className="border-b border-border last:border-0 hover:bg-surface-200/50">
                          <td className="px-4 py-2.5 text-sm font-mono text-white">{d.domain}</td>
                          <td className="px-4 py-2.5"><StatusBadge status={d.status === "active" ? "running" : d.status === "failed" ? "error" : "building"} /></td>
                          <td className="px-4 py-2.5 text-xs text-neutral-400">{d.tls_ready ? "Ready" : "Provisioning..."}</td>
                          <td className="px-4 py-2.5">
                            <button onClick={() => handleDeleteDomain(d.id)} className="p-1 text-red-400 hover:text-red-300 rounded transition-colors" title="Delete domain">
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>

            {/* Consumers — Coming Soon */}
            <section>
              <h2 className="mb-3 text-sm font-medium text-white">Consumers</h2>
              <div className="rounded-lg border border-border p-8 text-center">
                <p className="text-sm text-neutral-500">Coming Soon</p>
                <p className="mt-1 text-xs text-neutral-600">Consumer management (API keys, JWT credentials) will be available in Phase 2.</p>
              </div>
            </section>
          </>
        ) : null}
      </div>

      {/* ── Create Gateway Modal ── */}
      {showCreateGw && (
        <Modal title="Create Gateway" onClose={() => setShowCreateGw(false)}>
          <form onSubmit={e => { e.preventDefault(); handleCreateGw(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input type="text" value={newGwName} onChange={e => setNewGwName(e.target.value)} placeholder="My API Gateway"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" required autoFocus />
              <p className="mt-1 text-xs text-neutral-600">Slug will be auto-generated from name</p>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowCreateGw(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" disabled={creating} className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50">
                {creating ? "Creating..." : "Create Gateway"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* ── Add Route Modal ── */}
      {showAddRoute && (
        <Modal title={routeGroupId ? `Add Route to ${groupList.find(g => g.id === routeGroupId)?.name ?? "Group"}` : "Add Standalone Route"} onClose={() => setShowAddRoute(false)}>
          <form onSubmit={e => { e.preventDefault(); handleAddRoute(); }} className="space-y-3">
            <RouteFormFields
              name={routeName} setName={setRouteName}
              path={routePath} setPath={setRoutePath}
              methods={routeMethods} setMethods={setRouteMethods}
              appId={routeAppId} setAppId={setRouteAppId}
              gId={routeGroupId} setGId={setRouteGroupId}
              stripPrefix={routeStripPrefix} setStripPrefix={setRouteStripPrefix}
              authPoolId={routeAuthPoolId} setAuthPoolId={setRouteAuthPoolId}
              plugins={routePlugins} setPlugins={setRoutePlugins}
              isEdit={false}
            />
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowAddRoute(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" disabled={addingRoute || !addRouteValid} className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50">
                {addingRoute ? "Adding..." : "Add Route"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* ── Edit Route Modal ── */}
      {editRoute && (
        <Modal title={`Edit Route: ${editRoute.name}`} onClose={() => setEditRoute(null)}>
          <form onSubmit={e => { e.preventDefault(); handleUpdateRoute(); }} className="space-y-3">
            <RouteFormFields
              name={editRouteName} setName={setEditRouteName}
              path={editRoutePath} setPath={setEditRoutePath}
              methods={editRouteMethods} setMethods={setEditRouteMethods}
              appId={editRouteAppId} setAppId={setEditRouteAppId}
              gId={editRouteGroupId} setGId={setEditRouteGroupId}
              stripPrefix={editRouteStripPrefix} setStripPrefix={setEditRouteStripPrefix}
              authPoolId={editRouteAuthPoolId} setAuthPoolId={setEditRouteAuthPoolId}
              plugins={editRoutePlugins} setPlugins={setEditRoutePlugins}
              isEdit={true}
            />
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setEditRoute(null)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" disabled={savingRoute || !editRouteValid} className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50">
                {savingRoute ? "Saving..." : "Save Changes"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* ── Add Group Modal ── */}
      {showAddGroup && (
        <Modal title="New Group" onClose={() => setShowAddGroup(false)}>
          <form onSubmit={e => { e.preventDefault(); handleAddGroup(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input type="text" value={groupName} onChange={e => setGroupName(e.target.value)} placeholder="users-service"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" required autoFocus />
              <p className="mt-1 text-xs text-neutral-600">Groups bundle routes to the same app with shared plugins</p>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Target App</label>
              <select value={groupAppId} onChange={e => setGroupAppId(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none" required>
                <option value="">Select an app...</option>
                {userApps.map(app => <option key={app.id} value={app.id}>{app.name} ({app.subdomain})</option>)}
              </select>
            </div>
            <PluginEditor plugins={groupPlugins} onChange={setGroupPlugins} />
            <p className="text-xs text-neutral-600">Group plugins apply to all routes. Route-level plugins override on name collision.</p>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowAddGroup(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" disabled={addingGroup} className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50">
                {addingGroup ? "Creating..." : "Create Group"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* ── Edit Group Modal ── */}
      {editGroup && (
        <Modal title={`Edit Group: ${editGroup.name}`} onClose={() => setEditGroup(null)}>
          <form onSubmit={e => { e.preventDefault(); handleUpdateGroup(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input type="text" value={editGroupName} onChange={e => setEditGroupName(e.target.value)} placeholder="users-service"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" required autoFocus />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Target App</label>
              <select value={editGroupAppId} onChange={e => setEditGroupAppId(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none" required>
                <option value="">Select an app...</option>
                {userApps.map(app => <option key={app.id} value={app.id}>{app.name} ({app.subdomain})</option>)}
              </select>
            </div>
            <PluginEditor plugins={editGroupPlugins} onChange={setEditGroupPlugins} />
            <p className="text-xs text-neutral-600">Group plugins apply to all routes. Route-level plugins override on name collision.</p>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setEditGroup(null)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" disabled={savingGroup} className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50">
                {savingGroup ? "Saving..." : "Save Changes"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* ── Delete Gateway Confirmation ── */}
      {showDeleteGw && activeGw && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-xl border border-border bg-surface-100 p-6 shadow-xl">
            <div className="flex flex-col items-center text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
                <AlertTriangle className="h-6 w-6 text-red-400" />
              </div>
              <h3 className="mt-3 text-lg font-semibold text-white">Delete Gateway</h3>
              <p className="mt-1 text-xs text-neutral-500">This action cannot be undone</p>
            </div>
            <div className="mt-4 space-y-3">
              <p className="text-sm text-neutral-300">
                This will permanently delete <strong className="text-white">{activeGw.name}</strong> and all its routes, groups, and K8s resources.
              </p>
              <div>
                <label className="mb-1 block text-xs text-neutral-500">Type <strong className="text-white">{activeGw.name}</strong> to confirm</label>
                <input type="text" value={deleteGwConfirm} onChange={e => setDeleteGwConfirm(e.target.value)} autoFocus
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-red-500 focus:outline-none" />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => { setShowDeleteGw(false); setDeleteGwConfirm(""); }} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button onClick={handleDeleteGw} disabled={deleteGwConfirm !== activeGw.name || deletingGw}
                className="rounded-lg bg-red-500 px-4 py-2 text-sm font-medium text-white hover:bg-red-600 transition-colors disabled:opacity-50">
                {deletingGw ? "Deleting..." : "Delete Gateway"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Add Domain Modal ── */}
      {showAddDomain && (
        <Modal title="Add Custom Domain" onClose={() => setShowAddDomain(false)}>
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Domain</label>
              <input type="text" value={newDomain} onChange={e => setNewDomain(e.target.value)} placeholder="api.example.com" autoFocus
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" />
              <p className="mt-1 text-[10px] text-neutral-500">Point a CNAME record to your gateway endpoint before adding.</p>
            </div>
            <div className="flex justify-end gap-2">
              <button onClick={() => setShowAddDomain(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button onClick={handleAddDomain} disabled={addingDomain || !newDomain.trim()}
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50">
                {addingDomain ? "Adding..." : "Add Domain"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* ── Delete Group Confirmation ── */}
      {deleteGroupTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-xl border border-border bg-surface-100 p-6 shadow-xl">
            <div className="flex flex-col items-center text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
                <AlertTriangle className="h-6 w-6 text-red-400" />
              </div>
              <h3 className="mt-3 text-lg font-semibold text-white">Delete Group</h3>
              <p className="mt-1 text-xs text-neutral-500">This action cannot be undone</p>
            </div>
            <div className="mt-4 space-y-3">
              <p className="text-sm text-neutral-300">
                This will permanently delete the group <strong className="text-white">{deleteGroupTarget.name}</strong>. Routes in this group will become standalone.
              </p>
              <div>
                <label className="mb-1 block text-xs text-neutral-500">Type <strong className="text-white">{deleteGroupTarget.name}</strong> to confirm</label>
                <input type="text" value={deleteGroupConfirm} onChange={e => setDeleteGroupConfirm(e.target.value)} autoFocus
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-red-500 focus:outline-none" />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => { setDeleteGroupTarget(null); setDeleteGroupConfirm(""); }} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button onClick={() => handleDeleteGroup(deleteGroupTarget.id)} disabled={deleteGroupConfirm !== deleteGroupTarget.name || deletingGroup}
                className="rounded-lg bg-red-500 px-4 py-2 text-sm font-medium text-white hover:bg-red-600 transition-colors disabled:opacity-50">
                {deletingGroup ? "Deleting..." : "Delete Group"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Delete Route Confirmation ── */}
      {deleteRouteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-xl border border-border bg-surface-100 p-6 shadow-xl">
            <div className="flex flex-col items-center text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
                <AlertTriangle className="h-6 w-6 text-red-400" />
              </div>
              <h3 className="mt-3 text-lg font-semibold text-white">Delete Route</h3>
              <p className="mt-1 text-xs text-neutral-500">This action cannot be undone</p>
            </div>
            <p className="mt-4 text-sm text-neutral-300">
              Are you sure you want to delete the route <strong className="text-white">{deleteRouteTarget.name}</strong> ({deleteRouteTarget.path})?
            </p>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setDeleteRouteTarget(null)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button onClick={() => handleDeleteRoute(deleteRouteTarget.id)} disabled={deletingRoute}
                className="rounded-lg bg-red-500 px-4 py-2 text-sm font-medium text-white hover:bg-red-600 transition-colors disabled:opacity-50">
                {deletingRoute ? "Deleting..." : "Delete Route"}
              </button>
            </div>
          </div>
        </div>
      )}
    </Shell>
  );
}
