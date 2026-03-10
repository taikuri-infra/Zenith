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
import { type ApiGateway, type GatewayRouteInfo, type GatewayGroup, type DeployApp, type AuthPool } from "@/lib/api";
import { useState, useCallback } from "react";

const methodColors: Record<string, string> = {
  GET: "bg-emerald-500/10 text-emerald-400",
  POST: "bg-blue-500/10 text-blue-400",
  PUT: "bg-amber-500/10 text-amber-400",
  DELETE: "bg-red-500/10 text-red-400",
};

const AVAILABLE_PLUGINS = [
  { name: "cors", description: "Cross-Origin Resource Sharing" },
  { name: "limit-count", description: "Rate limiting by request count" },
  { name: "ip-restriction", description: "Allow/deny by IP address" },
  { name: "proxy-rewrite", description: "Rewrite upstream URI" },
  { name: "request-id", description: "Add unique request ID header" },
  { name: "key-auth", description: "API key authentication" },
  { name: "jwt-auth", description: "JWT authentication" },
];

// Default configs for common plugins
const DEFAULT_CONFIGS: Record<string, Record<string, unknown>> = {
  cors: { allow_origins: "*", allow_methods: "GET,POST,PUT,DELETE,OPTIONS", allow_headers: "*", max_age: 3600 },
  "limit-count": { count: 100, time_window: 60, rejected_code: 429 },
  "ip-restriction": { whitelist: ["0.0.0.0/0"] },
  "proxy-rewrite": { regex_uri: ["^/old(.*)", "/new$1"] },
  "request-id": { header_name: "X-Request-Id", include_in_response: true },
  "key-auth": {},
  "jwt-auth": {},
};

type PluginEntry = { name: string; enable: boolean; config: Record<string, unknown> };

function PluginEditor({ plugins, onChange }: { plugins: PluginEntry[]; onChange: (p: PluginEntry[]) => void }) {
  const [configErrors, setConfigErrors] = useState<Record<number, string>>({});

  const usedNames = new Set(plugins.map((p) => p.name));
  const available = AVAILABLE_PLUGINS.filter((p) => !usedNames.has(p.name));

  const addPlugin = (name: string) => {
    onChange([...plugins, { name, enable: true, config: DEFAULT_CONFIGS[name] ?? {} }]);
  };

  const removePlugin = (idx: number) => {
    onChange(plugins.filter((_, i) => i !== idx));
    setConfigErrors((prev) => {
      const next = { ...prev };
      delete next[idx];
      return next;
    });
  };

  const togglePlugin = (idx: number) => {
    onChange(plugins.map((p, i) => (i === idx ? { ...p, enable: !p.enable } : p)));
  };

  const updateConfig = (idx: number, raw: string) => {
    try {
      const parsed = JSON.parse(raw);
      onChange(plugins.map((p, i) => (i === idx ? { ...p, config: parsed } : p)));
      setConfigErrors((prev) => {
        const next = { ...prev };
        delete next[idx];
        return next;
      });
    } catch {
      setConfigErrors((prev) => ({ ...prev, [idx]: "Invalid JSON" }));
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="text-xs font-medium text-neutral-400">Plugins</label>
        {available.length > 0 && (
          <select
            value=""
            onChange={(e) => { if (e.target.value) addPlugin(e.target.value); }}
            className="rounded-md border border-border bg-surface-200 px-2 py-1 text-xs text-white focus:border-accent-500 focus:outline-none"
          >
            <option value="">+ Add plugin...</option>
            {available.map((p) => (
              <option key={p.name} value={p.name}>
                {p.name} — {p.description}
              </option>
            ))}
          </select>
        )}
      </div>
      {plugins.length === 0 && (
        <p className="text-xs text-neutral-600">No plugins configured</p>
      )}
      {plugins.map((plugin, idx) => (
        <div key={`${plugin.name}-${idx}`} className="rounded-md border border-border bg-surface-200 p-3 space-y-2">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => togglePlugin(idx)}
                className={`h-4 w-7 rounded-full transition-colors ${plugin.enable ? "bg-accent-500" : "bg-neutral-600"}`}
              >
                <span className={`block h-3 w-3 rounded-full bg-white transition-transform ${plugin.enable ? "translate-x-3.5" : "translate-x-0.5"}`} />
              </button>
              <span className="text-xs font-medium text-white">{plugin.name}</span>
            </div>
            <button
              type="button"
              onClick={() => removePlugin(idx)}
              className="text-xs text-red-400 hover:text-red-300"
            >
              Remove
            </button>
          </div>
          <div>
            <textarea
              defaultValue={JSON.stringify(plugin.config, null, 2)}
              onBlur={(e) => updateConfig(idx, e.target.value)}
              rows={3}
              className="w-full rounded-md border border-border bg-surface-300 px-2 py-1.5 font-mono text-xs text-neutral-300 placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              placeholder="{}"
            />
            {configErrors[idx] && (
              <p className="mt-0.5 text-xs text-red-400">{configErrors[idx]}</p>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

type Tab = "routes" | "groups";

export default function GatewayPage() {
  const { gateways, appsDeploy, authPools } = getApi();
  const projectId = useProject();

  // Gateways list
  const {
    data: gwList,
    loading: gwLoading,
    error: gwError,
    refetch: gwRefetch,
  } = useApi(() => gateways.list(projectId || undefined), [projectId]);

  // User's apps (for route/group target dropdown)
  const { data: appsData } = useApi(() => appsDeploy.list(projectId || undefined), [projectId]);
  const userApps: DeployApp[] = appsData?.items ?? [];

  // Auth pools (for route OIDC protection dropdown)
  const { data: poolsData } = useApi(() => authPools.list(), []);
  const pools: AuthPool[] = (poolsData ?? []).filter((p: AuthPool) => p.status === "active");

  // Selected gateway
  const [selectedGwId, setSelectedGwId] = useState<string | null>(null);
  const activeGw = gwList?.find((g: ApiGateway) => g.id === selectedGwId) ?? gwList?.[0] ?? null;

  // Active tab
  const [activeTab, setActiveTab] = useState<Tab>("routes");

  // Routes for selected gateway
  const {
    data: routes,
    loading: routesLoading,
    refetch: routesRefetch,
  } = useApi(
    () => activeGw ? gateways.listRoutes(activeGw.id) : Promise.resolve([]),
    [activeGw?.id]
  );

  // Groups for selected gateway
  const {
    data: groups,
    loading: groupsLoading,
    refetch: groupsRefetch,
  } = useApi(
    () => activeGw ? gateways.listGroups(activeGw.id) : Promise.resolve([]),
    [activeGw?.id]
  );

  // Create gateway modal
  const [showCreateGw, setShowCreateGw] = useState(false);
  const [newGwName, setNewGwName] = useState("");
  const [creating, setCreating] = useState(false);

  const handleCreateGw = useCallback(async () => {
    if (!newGwName.trim()) return;
    setCreating(true);
    try {
      const gw = await gateways.create(newGwName.trim());
      setShowCreateGw(false);
      setNewGwName("");
      setSelectedGwId(gw.id);
      gwRefetch();
    } catch {
      // error handled by useApi pattern
    } finally {
      setCreating(false);
    }
  }, [newGwName, gateways, gwRefetch]);

  // Delete gateway
  const handleDeleteGw = useCallback(async () => {
    if (!activeGw) return;
    if (!confirm(`Delete gateway "${activeGw.name}"? This removes all routes and K8s resources.`)) return;
    try {
      await gateways.delete(activeGw.id);
      setSelectedGwId(null);
      gwRefetch();
    } catch {
      // handled
    }
  }, [activeGw, gateways, gwRefetch]);

  // Sync gateway
  const handleSyncGw = useCallback(async () => {
    if (!activeGw) return;
    try {
      await gateways.sync(activeGw.id);
      routesRefetch();
      groupsRefetch();
    } catch {
      // handled
    }
  }, [activeGw, gateways, routesRefetch, groupsRefetch]);

  // --- Route state ---
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

  const handleAddRoute = useCallback(async () => {
    if (!activeGw || !routeName.trim() || !routePath.trim()) return;
    if (!routeAppId && !routeGroupId) return;
    const methods = Object.entries(routeMethods).filter(([, v]) => v).map(([k]) => k);
    if (methods.length === 0) methods.push("GET");

    setAddingRoute(true);
    try {
      await gateways.createRoute(activeGw.id, {
        name: routeName.trim(),
        path: routePath.trim(),
        methods,
        ...(routeGroupId ? { group_id: routeGroupId } : { app_id: routeAppId }),
        strip_prefix: routeStripPrefix,
        ...(routeAuthPoolId ? { auth_pool_id: routeAuthPoolId } : {}),
        ...(routePlugins.length > 0 ? { plugins: routePlugins } : {}),
      });
      setShowAddRoute(false);
      setRouteName("");
      setRoutePath("");
      setRouteAppId("");
      setRouteGroupId("");
      setRouteMethods({ GET: true });
      setRouteStripPrefix(false);
      setRouteAuthPoolId("");
      setRoutePlugins([]);
      routesRefetch();
      gwRefetch();
    } catch {
      // handled
    } finally {
      setAddingRoute(false);
    }
  }, [activeGw, routeName, routePath, routeAppId, routeGroupId, routeMethods, routeStripPrefix, routeAuthPoolId, gateways, routesRefetch, gwRefetch]);

  // Delete route
  const handleDeleteRoute = useCallback(async (routeId: string) => {
    if (!activeGw) return;
    try {
      await gateways.deleteRoute(activeGw.id, routeId);
      routesRefetch();
      gwRefetch();
    } catch {
      // handled
    }
  }, [activeGw, gateways, routesRefetch, gwRefetch]);

  // --- Group state ---
  const [showAddGroup, setShowAddGroup] = useState(false);
  const [groupName, setGroupName] = useState("");
  const [groupAppId, setGroupAppId] = useState("");
  const [groupPlugins, setGroupPlugins] = useState<PluginEntry[]>([]);
  const [addingGroup, setAddingGroup] = useState(false);

  const handleAddGroup = useCallback(async () => {
    if (!activeGw || !groupName.trim() || !groupAppId) return;
    setAddingGroup(true);
    try {
      await gateways.createGroup(activeGw.id, {
        name: groupName.trim(),
        app_id: groupAppId,
        ...(groupPlugins.length > 0 ? { plugins: groupPlugins } : {}),
      });
      setShowAddGroup(false);
      setGroupName("");
      setGroupAppId("");
      setGroupPlugins([]);
      groupsRefetch();
    } catch {
      // handled
    } finally {
      setAddingGroup(false);
    }
  }, [activeGw, groupName, groupAppId, gateways, groupsRefetch]);

  const handleDeleteGroup = useCallback(async (groupId: string) => {
    if (!activeGw) return;
    try {
      await gateways.deleteGroup(activeGw.id, groupId);
      groupsRefetch();
      routesRefetch();
    } catch {
      // handled
    }
  }, [activeGw, gateways, groupsRefetch, routesRefetch]);

  if (gwLoading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={6} rows={4} />
      </Shell>
    );
  }

  if (gwError) {
    return (
      <Shell>
        <ErrorState message={gwError} onRetry={gwRefetch} />
      </Shell>
    );
  }

  const gatewayList: ApiGateway[] = gwList ?? [];
  const routeList: GatewayRouteInfo[] = routes ?? [];
  const groupList: GatewayGroup[] = groups ?? [];

  // Helper to find group name by id
  const groupName4Route = (groupId?: string) => {
    if (!groupId) return null;
    return groupList.find(g => g.id === groupId)?.name ?? null;
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">API Gateway</h1>
            <p className="text-sm text-neutral-500">APISIX-powered traffic management and routing</p>
          </div>
          <button
            onClick={() => setShowCreateGw(true)}
            className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
          >
            + New Gateway
          </button>
        </div>

        {/* Gateway Selector */}
        {gatewayList.length > 0 && (
          <div className="flex items-center gap-3">
            <label className="text-xs font-medium text-neutral-400">Gateway:</label>
            <select
              value={activeGw?.id ?? ""}
              onChange={(e) => setSelectedGwId(e.target.value)}
              className="rounded-md border border-border bg-surface-200 px-3 py-1.5 text-sm text-white focus:border-accent-500 focus:outline-none"
            >
              {gatewayList.map((gw) => (
                <option key={gw.id} value={gw.id}>
                  {gw.name} ({gw.slug})
                </option>
              ))}
            </select>
            {activeGw && (
              <>
                <StatusBadge status={activeGw.status === "active" ? "running" : activeGw.status === "error" ? "error" : "pending"} />
                <span className="font-mono text-xs text-neutral-500">{activeGw.endpoint}</span>
                <button
                  onClick={handleSyncGw}
                  className="ml-auto rounded-md border border-border px-2.5 py-1 text-xs text-neutral-400 hover:text-white transition-colors"
                  title="Force reconcile K8s CRDs"
                >
                  Sync
                </button>
                <button
                  onClick={handleDeleteGw}
                  className="rounded-md border border-red-500/30 px-2.5 py-1 text-xs text-red-400 hover:bg-red-500/10 transition-colors"
                >
                  Delete
                </button>
              </>
            )}
          </div>
        )}

        {gatewayList.length === 0 ? (
          <EmptyState
            title="No gateways yet"
            description="Create your first API gateway to route traffic to your apps with plugins."
            actionLabel="Create Gateway"
            onAction={() => setShowCreateGw(true)}
          />
        ) : activeGw ? (
          <>
            {/* Stats */}
            <div className="grid grid-cols-4 gap-4">
              <StatCard label="Endpoint" value={activeGw.slug + ".gw.*"} />
              <StatCard label="Status" value={activeGw.status} />
              <StatCard label="Active Routes" value={String(routeList.filter(r => r.status === "active").length)} />
              <StatCard label="Groups" value={String(groupList.length)} />
            </div>

            {/* Tabs */}
            <div className="border-b border-border">
              <div className="flex gap-6">
                {(["routes", "groups"] as Tab[]).map((tab) => (
                  <button
                    key={tab}
                    onClick={() => setActiveTab(tab)}
                    className={`pb-2 text-sm font-medium transition-colors ${
                      activeTab === tab
                        ? "border-b-2 border-accent-500 text-white"
                        : "text-neutral-500 hover:text-neutral-300"
                    }`}
                  >
                    {tab === "routes" ? `Routes (${routeList.length})` : `Groups (${groupList.length})`}
                  </button>
                ))}
              </div>
            </div>

            {/* Routes Tab */}
            {activeTab === "routes" && (
              <section>
                <div className="mb-3 flex items-center justify-between">
                  <h2 className="text-sm font-medium text-white">Routes</h2>
                  <button
                    onClick={() => setShowAddRoute(true)}
                    className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
                  >
                    + Add Route
                  </button>
                </div>

                {routesLoading ? (
                  <div className="rounded-lg border border-border p-8 text-center text-sm text-neutral-500">Loading routes...</div>
                ) : routeList.length === 0 ? (
                  <div className="rounded-lg border border-border p-8 text-center text-sm text-neutral-500">
                    No routes yet. Add a route to start routing traffic.
                  </div>
                ) : (
                  <div className="overflow-hidden rounded-lg border border-border">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-border bg-surface-100">
                          <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Route</th>
                          <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Path</th>
                          <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Methods</th>
                          <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Target</th>
                          <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Group</th>
                          <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Plugins</th>
                          <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                          <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500"></th>
                        </tr>
                      </thead>
                      <tbody>
                        {routeList.map((route) => (
                          <tr key={route.id} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                            <td className="px-4 py-3 font-medium text-white">{route.name}</td>
                            <td className="px-4 py-3 font-mono text-xs text-neutral-400">{route.path}</td>
                            <td className="px-4 py-3">
                              <div className="flex flex-wrap gap-1">
                                {route.methods.map((method) => (
                                  <span
                                    key={method}
                                    className={`inline-flex rounded px-1.5 py-0.5 text-[10px] font-semibold ${methodColors[method] || "bg-neutral-500/10 text-neutral-400"}`}
                                  >
                                    {method}
                                  </span>
                                ))}
                              </div>
                            </td>
                            <td className="px-4 py-3 font-mono text-xs text-neutral-400">
                              {route.app_subdomain || (route.group_id ? "(from group)" : "-")}
                            </td>
                            <td className="px-4 py-3 text-xs text-neutral-400">
                              {groupName4Route(route.group_id) ? (
                                <span className="inline-flex rounded bg-cyan-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-cyan-400">
                                  {groupName4Route(route.group_id)}
                                </span>
                              ) : (
                                <span className="text-neutral-600">-</span>
                              )}
                            </td>
                            <td className="px-4 py-3">
                              <div className="flex flex-wrap gap-1">
                                {route.auth === "oidc" && (
                                  <span className="inline-flex rounded bg-purple-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-purple-400">
                                    OIDC
                                  </span>
                                )}
                                {route.plugins.map((plugin, i) => (
                                  <span key={`${plugin.name}-${i}`} className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-400">
                                    {plugin.name}
                                  </span>
                                ))}
                                {route.strip_prefix && (
                                  <span className="inline-flex rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] text-amber-400">
                                    strip-prefix
                                  </span>
                                )}
                              </div>
                            </td>
                            <td className="px-4 py-3">
                              <StatusBadge status={route.status === "active" ? "running" : "stopped"} />
                            </td>
                            <td className="px-4 py-3">
                              <button
                                onClick={() => handleDeleteRoute(route.id)}
                                className="text-xs text-red-400 hover:text-red-300 transition-colors"
                              >
                                Delete
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </section>
            )}

            {/* Groups Tab */}
            {activeTab === "groups" && (
              <section>
                <div className="mb-3 flex items-center justify-between">
                  <h2 className="text-sm font-medium text-white">Groups</h2>
                  <button
                    onClick={() => setShowAddGroup(true)}
                    className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
                  >
                    + Add Group
                  </button>
                </div>

                {groupsLoading ? (
                  <div className="rounded-lg border border-border p-8 text-center text-sm text-neutral-500">Loading groups...</div>
                ) : groupList.length === 0 ? (
                  <div className="rounded-lg border border-border p-8 text-center text-sm text-neutral-500">
                    No groups yet. Groups bundle routes pointing to the same app with shared plugins.
                  </div>
                ) : (
                  <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {groupList.map((group) => {
                      const routeCount = routeList.filter(r => r.group_id === group.id).length;
                      return (
                        <div key={group.id} className="rounded-lg border border-border bg-surface-100 p-4 space-y-3">
                          <div className="flex items-center justify-between">
                            <h3 className="text-sm font-medium text-white">{group.name}</h3>
                            <button
                              onClick={() => handleDeleteGroup(group.id)}
                              className="text-xs text-red-400 hover:text-red-300 transition-colors"
                            >
                              Delete
                            </button>
                          </div>
                          <div className="space-y-1.5 text-xs">
                            <div className="flex items-center justify-between">
                              <span className="text-neutral-500">Target App</span>
                              <span className="font-mono text-neutral-300">{group.app_subdomain}</span>
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-neutral-500">Routes</span>
                              <span className="text-neutral-300">{routeCount}</span>
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-neutral-500">Plugins</span>
                              <span className="text-neutral-300">{group.plugins.length}</span>
                            </div>
                          </div>
                          {group.plugins.length > 0 && (
                            <div className="flex flex-wrap gap-1">
                              {group.plugins.map((plugin, i) => (
                                <span key={`${plugin.name}-${i}`} className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-400">
                                  {plugin.name}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </section>
            )}

            {/* Consumers — Coming Soon */}
            <section>
              <div className="mb-3">
                <h2 className="text-sm font-medium text-white">Consumers</h2>
              </div>
              <div className="rounded-lg border border-border p-8 text-center">
                <p className="text-sm text-neutral-500">Coming Soon</p>
                <p className="mt-1 text-xs text-neutral-600">Consumer management (API keys, JWT credentials) will be available in Phase 2.</p>
              </div>
            </section>
          </>
        ) : null}
      </div>

      {/* Create Gateway Modal */}
      {showCreateGw && (
        <Modal title="Create Gateway" onClose={() => setShowCreateGw(false)}>
          <form
            onSubmit={(e) => { e.preventDefault(); handleCreateGw(); }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input
                type="text"
                value={newGwName}
                onChange={(e) => setNewGwName(e.target.value)}
                placeholder="My API Gateway"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
                autoFocus
              />
              <p className="mt-1 text-xs text-neutral-600">
                Slug will be auto-generated from name (e.g. &quot;my-api-gateway&quot;)
              </p>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => setShowCreateGw(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating}
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {creating ? "Creating..." : "Create Gateway"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Add Route Modal */}
      {showAddRoute && (
        <Modal title="Add Route" onClose={() => setShowAddRoute(false)}>
          <form
            onSubmit={(e) => { e.preventDefault(); handleAddRoute(); }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input
                type="text"
                value={routeName}
                onChange={(e) => setRouteName(e.target.value)}
                placeholder="users-api"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Path</label>
              <input
                type="text"
                value={routePath}
                onChange={(e) => setRoutePath(e.target.value)}
                placeholder="/api/v1/users/*"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            {groupList.length > 0 && (
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Group (optional)</label>
                <select
                  value={routeGroupId}
                  onChange={(e) => {
                    setRouteGroupId(e.target.value);
                    if (e.target.value) setRouteAppId(""); // clear app when group selected
                  }}
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                >
                  <option value="">Standalone (no group)</option>
                  {groupList.map((g) => (
                    <option key={g.id} value={g.id}>
                      {g.name} ({g.app_subdomain})
                    </option>
                  ))}
                </select>
                <p className="mt-1 text-xs text-neutral-600">
                  Group routes inherit the target app and shared plugins from the group
                </p>
              </div>
            )}
            {!routeGroupId && (
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Target App</label>
                <select
                  value={routeAppId}
                  onChange={(e) => setRouteAppId(e.target.value)}
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                  required={!routeGroupId}
                >
                  <option value="">Select an app...</option>
                  {userApps.map((app) => (
                    <option key={app.id} value={app.id}>
                      {app.name} ({app.subdomain})
                    </option>
                  ))}
                </select>
              </div>
            )}
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Methods</label>
              <div className="flex gap-4 pt-1">
                {["GET", "POST", "PUT", "DELETE"].map((method) => (
                  <label key={method} className="flex items-center gap-1.5 text-xs text-neutral-300">
                    <input
                      type="checkbox"
                      checked={routeMethods[method] || false}
                      onChange={(e) => setRouteMethods((prev) => ({ ...prev, [method]: e.target.checked }))}
                      className="rounded border-border bg-surface-200"
                    />
                    {method}
                  </label>
                ))}
              </div>
            </div>
            <div>
              <label className="flex items-center gap-2 text-xs text-neutral-300">
                <input
                  type="checkbox"
                  checked={routeStripPrefix}
                  onChange={(e) => setRouteStripPrefix(e.target.checked)}
                  className="rounded border-border bg-surface-200"
                />
                Strip path prefix before forwarding
              </label>
            </div>
            {pools.length > 0 && (
              <div>
                <label className="mb-1 block text-xs font-medium text-neutral-400">Auth Pool (OIDC)</label>
                <select
                  value={routeAuthPoolId}
                  onChange={(e) => setRouteAuthPoolId(e.target.value)}
                  className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                >
                  <option value="">None (no authentication)</option>
                  {pools.map((pool) => (
                    <option key={pool.id} value={pool.id}>
                      {pool.name} ({pool.user_count} users)
                    </option>
                  ))}
                </select>
                <p className="mt-1 text-xs text-neutral-600">
                  Protect this route with OIDC — APISIX auto-validates JWT bearer tokens
                </p>
              </div>
            )}
            <PluginEditor plugins={routePlugins} onChange={setRoutePlugins} />
            {routeGroupId && (
              <p className="text-xs text-cyan-400">
                Route-level plugins override group plugins with the same name
              </p>
            )}
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => setShowAddRoute(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={addingRoute}
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {addingRoute ? "Adding..." : "Add Route"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Add Group Modal */}
      {showAddGroup && (
        <Modal title="Add Group" onClose={() => setShowAddGroup(false)}>
          <form
            onSubmit={(e) => { e.preventDefault(); handleAddGroup(); }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input
                type="text"
                value={groupName}
                onChange={(e) => setGroupName(e.target.value)}
                placeholder="users-service"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
                autoFocus
              />
              <p className="mt-1 text-xs text-neutral-600">
                Groups bundle multiple routes pointing to the same app with shared plugins
              </p>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Target App</label>
              <select
                value={groupAppId}
                onChange={(e) => setGroupAppId(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
                required
              >
                <option value="">Select an app...</option>
                {userApps.map((app) => (
                  <option key={app.id} value={app.id}>
                    {app.name} ({app.subdomain})
                  </option>
                ))}
              </select>
            </div>
            <PluginEditor plugins={groupPlugins} onChange={setGroupPlugins} />
            <p className="text-xs text-neutral-600">
              Group plugins apply to all routes in the group. Route-level plugins override on name collision.
            </p>
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => setShowAddGroup(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={addingGroup}
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {addingGroup ? "Adding..." : "Add Group"}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
