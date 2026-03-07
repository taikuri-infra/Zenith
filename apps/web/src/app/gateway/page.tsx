"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { StatCard } from "@/components/stat-card";
import { Modal } from "@/components/modal";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { useApi } from "@/hooks/use-api";
import { getApi } from "@/lib/get-api";
import { type ApiGateway, type GatewayRouteInfo, type DeployApp } from "@/lib/api";
import { useState, useCallback } from "react";

const methodColors: Record<string, string> = {
  GET: "bg-emerald-500/10 text-emerald-400",
  POST: "bg-blue-500/10 text-blue-400",
  PUT: "bg-amber-500/10 text-amber-400",
  DELETE: "bg-red-500/10 text-red-400",
};

export default function GatewayPage() {
  const { gateways, appsDeploy } = getApi();

  // Gateways list
  const {
    data: gwList,
    loading: gwLoading,
    error: gwError,
    refetch: gwRefetch,
  } = useApi(() => gateways.list(), []);

  // User's apps (for route target dropdown)
  const { data: appsData } = useApi(() => appsDeploy.list(), []);
  const userApps: DeployApp[] = appsData?.items ?? [];

  // Selected gateway
  const [selectedGwId, setSelectedGwId] = useState<string | null>(null);
  const activeGw = gwList?.find((g: ApiGateway) => g.id === selectedGwId) ?? gwList?.[0] ?? null;

  // Routes for selected gateway
  const {
    data: routes,
    loading: routesLoading,
    refetch: routesRefetch,
  } = useApi(
    () => activeGw ? gateways.listRoutes(activeGw.id) : Promise.resolve([]),
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
    } catch {
      // handled
    }
  }, [activeGw, gateways, routesRefetch]);

  // Add route modal
  const [showAddRoute, setShowAddRoute] = useState(false);
  const [routeName, setRouteName] = useState("");
  const [routePath, setRoutePath] = useState("");
  const [routeAppId, setRouteAppId] = useState("");
  const [routeMethods, setRouteMethods] = useState<Record<string, boolean>>({ GET: true });
  const [routeStripPrefix, setRouteStripPrefix] = useState(false);
  const [addingRoute, setAddingRoute] = useState(false);

  const handleAddRoute = useCallback(async () => {
    if (!activeGw || !routeName.trim() || !routePath.trim() || !routeAppId) return;
    const methods = Object.entries(routeMethods).filter(([, v]) => v).map(([k]) => k);
    if (methods.length === 0) methods.push("GET");

    setAddingRoute(true);
    try {
      await gateways.createRoute(activeGw.id, {
        name: routeName.trim(),
        path: routePath.trim(),
        methods,
        app_id: routeAppId,
        strip_prefix: routeStripPrefix,
      });
      setShowAddRoute(false);
      setRouteName("");
      setRoutePath("");
      setRouteAppId("");
      setRouteMethods({ GET: true });
      setRouteStripPrefix(false);
      routesRefetch();
      gwRefetch();
    } catch {
      // handled
    } finally {
      setAddingRoute(false);
    }
  }, [activeGw, routeName, routePath, routeAppId, routeMethods, routeStripPrefix, gateways, routesRefetch, gwRefetch]);

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
              <StatCard label="Total Routes" value={String(activeGw.route_count)} />
            </div>

            {/* Routes */}
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
                        <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Target App</th>
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
                          <td className="px-4 py-3 font-mono text-xs text-neutral-400">{route.app_subdomain}</td>
                          <td className="px-4 py-3">
                            <div className="flex flex-wrap gap-1">
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
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Target App</label>
              <select
                value={routeAppId}
                onChange={(e) => setRouteAppId(e.target.value)}
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
    </Shell>
  );
}
