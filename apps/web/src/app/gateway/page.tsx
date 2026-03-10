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
import { getApi } from "@/lib/get-api";
import { type ApiGateway, type GatewayRouteInfo, type GatewayGroup, type GatewayRoutePlugin, type DeployApp, type AuthPool, type CreateRouteInput } from "@/lib/api";
import { useState, useCallback } from "react";

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

/* ── plugin editor component (form fields, NOT raw JSON) ── */
function PluginEditor({ plugins, onChange }: { plugins: PluginEntry[]; onChange: (p: PluginEntry[]) => void }) {
  const usedNames = new Set(plugins.map(p => p.name));
  const available = Object.entries(PLUGIN_DEFS).filter(([n]) => !usedNames.has(n));

  const update = (idx: number, key: string, value: unknown) =>
    onChange(plugins.map((p, i) => i === idx ? { ...p, config: { ...p.config, [key]: value } } : p));

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="text-xs font-medium text-neutral-400">Plugins</label>
        {available.length > 0 && (
          <select
            value=""
            onChange={e => { if (e.target.value) onChange([...plugins, { name: e.target.value, enable: true, config: { ...PLUGIN_DEFS[e.target.value].defaults } }]); }}
            className="rounded-md border border-border bg-surface-200 px-2 py-1 text-xs text-white focus:border-accent-500 focus:outline-none"
          >
            <option value="">+ Add plugin...</option>
            {available.map(([n, d]) => <option key={n} value={n}>{n} — {d.description}</option>)}
          </select>
        )}
      </div>
      {plugins.length === 0 && <p className="text-xs text-neutral-600">No plugins configured</p>}
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
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════ */
/*  Main Page                                                    */
/* ══════════════════════════════════════════════════════════════ */

export default function GatewayPage() {
  const { gateways, appsDeploy, authPools } = getApi();
  const projectId = useProject();

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
    } finally { setCreating(false); }
  }, [newGwName, gateways, gwRefetch]);

  const handleDeleteGw = useCallback(async () => {
    if (!activeGw || !confirm(`Delete gateway "${activeGw.name}"? This removes all routes and K8s resources.`)) return;
    await gateways.delete(activeGw.id);
    setSelectedGwId(null); gwRefetch();
  }, [activeGw, gateways, gwRefetch]);

  const handleSyncGw = useCallback(async () => {
    if (!activeGw) return;
    await gateways.sync(activeGw.id);
    routesRefetch(); groupsRefetch();
  }, [activeGw, gateways, routesRefetch, groupsRefetch]);

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
    } finally { setAddingRoute(false); }
  }, [activeGw, routeName, routePath, routeAppId, routeGroupId, routeMethods, routeStripPrefix, routeAuthPoolId, routePlugins, gateways, routesRefetch, gwRefetch]);

  const handleDeleteRoute = useCallback(async (routeId: string) => {
    if (!activeGw) return;
    await gateways.deleteRoute(activeGw.id, routeId);
    routesRefetch(); gwRefetch();
  }, [activeGw, gateways, routesRefetch, gwRefetch]);

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
    } finally { setAddingGroup(false); }
  }, [activeGw, groupName, groupAppId, groupPlugins, gateways, groupsRefetch]);

  const handleDeleteGroup = useCallback(async (groupId: string) => {
    if (!activeGw) return;
    await gateways.deleteGroup(activeGw.id, groupId);
    groupsRefetch(); routesRefetch();
  }, [activeGw, gateways, groupsRefetch, routesRefetch]);

  /* ── Render ── */

  if (gwLoading) return <Shell><PageWithTableSkeleton cols={6} rows={4} /></Shell>;
  if (gwError) return <Shell><ErrorState message={gwError} onRetry={gwRefetch} /></Shell>;

  const gatewayList: ApiGateway[] = gwList ?? [];
  const standaloneRoutes = routeList.filter(r => !r.group_id);

  const routeRow = (route: GatewayRouteInfo) => (
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
          {route.plugins.map((p, i) => (
            <span key={`${p.name}-${i}`} className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-400">{p.name}</span>
          ))}
          {route.auth === "oidc" && <span className="inline-flex rounded bg-purple-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-purple-400">OIDC</span>}
          {route.strip_prefix && <span className="inline-flex rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] text-amber-400">strip-prefix</span>}
        </div>
      </td>
      <td className="px-4 py-2.5"><StatusBadge status={route.status === "active" ? "running" : "stopped"} /></td>
      <td className="px-4 py-2.5">
        <button onClick={() => handleDeleteRoute(route.id)} className="text-xs text-red-400 hover:text-red-300 transition-colors">Delete</button>
      </td>
    </tr>
  );

  const routeTableHead = (
    <thead>
      <tr className="border-b border-border bg-surface-100/50">
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Name</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Path</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Methods</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Plugins</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500">Status</th>
        <th className="px-4 py-2 text-left text-xs font-medium text-neutral-500"></th>
      </tr>
    </thead>
  );

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
                <button onClick={handleDeleteGw} className="rounded-md border border-red-500/30 px-2.5 py-1 text-xs text-red-400 hover:bg-red-500/10 transition-colors">Delete</button>
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
                            onClick={e => { e.stopPropagation(); handleDeleteGroup(group.id); }}
                            className="ml-2 text-xs text-red-400 hover:text-red-300 transition-colors"
                          >Delete</button>
                        </div>
                      </button>

                      {/* Expanded: routes table + add route */}
                      {isExpanded && (
                        <div className="border-t border-border">
                          {groupRoutes.length > 0 ? (
                            <table className="w-full text-sm">
                              {routeTableHead}
                              <tbody>{groupRoutes.map(routeRow)}</tbody>
                            </table>
                          ) : (
                            <div className="px-4 py-4 text-center text-xs text-neutral-600">
                              No routes in this group yet
                            </div>
                          )}
                          <div className="border-t border-border px-4 py-2">
                            <button onClick={() => openAddRoute(group.id)} className="text-xs text-accent-400 hover:text-accent-300 transition-colors">
                              + Add Route to {group.name}
                            </button>
                          </div>
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
                        <tbody>{standaloneRoutes.map(routeRow)}</tbody>
                      </table>
                    </div>
                  </div>
                )}

                {groupList.length === 0 && standaloneRoutes.length === 0 && (
                  <div className="rounded-lg border border-border p-8 text-center text-sm text-neutral-500">
                    No groups or routes yet. Start by creating a group, then add routes to it.
                  </div>
                )}
              </div>
            )}

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
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input type="text" value={routeName} onChange={e => setRouteName(e.target.value)} placeholder="users-api"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" required autoFocus />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Path</label>
              <input type="text" value={routePath} onChange={e => setRoutePath(e.target.value)} placeholder="/api/v1/users/*"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none" required />
            </div>

            {/* Group selector — only show if creating standalone (not pre-assigned) */}
            {!routeGroupId && groupList.length > 0 && (
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Group (optional)</label>
                <select value={routeGroupId} onChange={e => { setRouteGroupId(e.target.value); if (e.target.value) setRouteAppId(""); }}
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none">
                  <option value="">Standalone (select target app below)</option>
                  {groupList.map(g => <option key={g.id} value={g.id}>{g.name} ({g.app_subdomain})</option>)}
                </select>
              </div>
            )}

            {/* Target app — only when standalone */}
            {!routeGroupId && (
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Target App</label>
                <select value={routeAppId} onChange={e => setRouteAppId(e.target.value)}
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none" required={!routeGroupId}>
                  <option value="">Select an app...</option>
                  {userApps.map(app => <option key={app.id} value={app.id}>{app.name} ({app.subdomain})</option>)}
                </select>
              </div>
            )}

            {routeGroupId && (
              <div className="rounded-md bg-accent-500/5 border border-accent-500/20 px-3 py-2">
                <p className="text-xs text-accent-400">Target app inherited from group. Route-level plugins override group plugins with the same name.</p>
              </div>
            )}

            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Methods</label>
              <div className="flex gap-4 pt-1">
                {["GET", "POST", "PUT", "DELETE"].map(m => (
                  <label key={m} className="flex items-center gap-1.5 text-xs text-neutral-300">
                    <input type="checkbox" checked={routeMethods[m] || false} onChange={e => setRouteMethods(prev => ({ ...prev, [m]: e.target.checked }))} className="rounded border-border bg-surface-200" />
                    {m}
                  </label>
                ))}
              </div>
            </div>
            <label className="flex items-center gap-2 text-xs text-neutral-300">
              <input type="checkbox" checked={routeStripPrefix} onChange={e => setRouteStripPrefix(e.target.checked)} className="rounded border-border bg-surface-200" />
              Strip path prefix before forwarding
            </label>
            {pools.length > 0 && (
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Auth Pool (OIDC)</label>
                <select value={routeAuthPoolId} onChange={e => setRouteAuthPoolId(e.target.value)}
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none">
                  <option value="">None</option>
                  {pools.map(pool => <option key={pool.id} value={pool.id}>{pool.name} ({pool.user_count} users)</option>)}
                </select>
              </div>
            )}
            <PluginEditor plugins={routePlugins} onChange={setRoutePlugins} />
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowAddRoute(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" disabled={addingRoute} className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50">
                {addingRoute ? "Adding..." : "Add Route"}
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
    </Shell>
  );
}
