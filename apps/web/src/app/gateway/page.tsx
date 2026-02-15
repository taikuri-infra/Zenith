"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { StatCard } from "@/components/stat-card";
import { Modal } from "@/components/modal";
import { useState } from "react";

interface Route {
  name: string;
  path: string;
  methods: string[];
  service: string;
  plugins: string[];
  reqMin: string;
  latency: string;
  status: "running" | "stopped";
}

interface Plugin {
  name: string;
  scope: string;
  appliedTo: string;
  config: string;
  enabled: boolean;
}

const initialRoutes: Route[] = [
  { name: "users-api", path: "/api/v1/users/*", methods: ["GET", "POST", "PUT", "DELETE"], service: "user-service:8080", plugins: ["jwt-auth", "rate-limit"], reqMin: "1,850", latency: "23ms", status: "running" },
  { name: "orders-api", path: "/api/v1/orders/*", methods: ["GET", "POST"], service: "order-service:8080", plugins: ["jwt-auth", "rate-limit"], reqMin: "920", latency: "34ms", status: "running" },
  { name: "payments-api", path: "/api/v1/payments/*", methods: ["POST"], service: "payment-service:8080", plugins: ["jwt-auth", "rate-limit", "request-transform"], reqMin: "340", latency: "67ms", status: "running" },
  { name: "auth-api", path: "/api/v1/auth/*", methods: ["GET", "POST"], service: "auth-service:8080", plugins: ["rate-limit", "cors"], reqMin: "4,280", latency: "12ms", status: "running" },
  { name: "notifications", path: "/api/v1/notifications/*", methods: ["POST"], service: "notification-svc:8080", plugins: ["jwt-auth"], reqMin: "0", latency: "\u2014", status: "stopped" },
  { name: "webhooks", path: "/webhooks/*", methods: ["POST"], service: "webhook-handler:8080", plugins: ["ip-restrict", "hmac-auth"], reqMin: "120", latency: "8ms", status: "running" },
  { name: "frontend", path: "/*", methods: ["GET"], service: "frontend:3000", plugins: ["cors"], reqMin: "2,140", latency: "45ms", status: "running" },
];

const initialPlugins: Plugin[] = [
  { name: "jwt-auth", scope: "global", appliedTo: "All routes", config: "issuer: auth.startup.zenith.cloud", enabled: true },
  { name: "rate-limiting", scope: "global", appliedTo: "All routes", config: "1000 req/min per consumer", enabled: true },
  { name: "cors", scope: "global", appliedTo: "All routes", config: "origins: *.startup.com", enabled: true },
  { name: "request-transformer", scope: "route", appliedTo: "payments-api", config: "add-header: X-Payment-Version=v2", enabled: true },
  { name: "ip-restriction", scope: "route", appliedTo: "webhooks", config: "allow: 104.18.0.0/16, 172.64.0.0/13", enabled: true },
  { name: "bot-detection", scope: "global", appliedTo: "All routes", config: "block: scrapers, crawlers", enabled: false },
];

const mockConsumers = [
  { consumer: "web-app", username: "web-app-consumer", credentials: "JWT + API Key", created: "Nov 1, 2025", requests24h: "892K" },
  { consumer: "mobile-app", username: "mobile-consumer", credentials: "JWT", created: "Dec 15, 2025", requests24h: "234K" },
  { consumer: "partner-api", username: "partner-consumer", credentials: "JWT + API Key", created: "Jan 20, 2026", requests24h: "45K" },
];

const methodColors: Record<string, string> = {
  GET: "bg-emerald-500/10 text-emerald-400",
  POST: "bg-blue-500/10 text-blue-400",
  PUT: "bg-amber-500/10 text-amber-400",
  DELETE: "bg-red-500/10 text-red-400",
};

export default function GatewayPage() {
  const [routes, setRoutes] = useState<Route[]>(initialRoutes);
  const [plugins, setPlugins] = useState<Plugin[]>(initialPlugins);

  const [showAddRoute, setShowAddRoute] = useState(false);
  const [routeName, setRouteName] = useState("");
  const [routePath, setRoutePath] = useState("");
  const [routeService, setRouteService] = useState("");
  const [routeMethods, setRouteMethods] = useState<Record<string, boolean>>({ GET: false, POST: false, PUT: false, DELETE: false });

  const [showAddPlugin, setShowAddPlugin] = useState(false);
  const [pluginName, setPluginName] = useState("jwt-auth");
  const [pluginScope, setPluginScope] = useState("global");
  const [pluginAppliedTo, setPluginAppliedTo] = useState("");

  const handleAddRoute = () => {
    if (!routeName.trim() || !routePath.trim()) return;
    const selectedMethods = Object.entries(routeMethods).filter(([, v]) => v).map(([k]) => k);
    if (selectedMethods.length === 0) selectedMethods.push("GET");
    const newRoute: Route = {
      name: routeName.trim(),
      path: routePath.trim(),
      methods: selectedMethods,
      service: routeService.trim() || "unknown:8080",
      plugins: [],
      reqMin: "0",
      latency: "\u2014",
      status: "running",
    };
    setRoutes((prev) => [...prev, newRoute]);
    setShowAddRoute(false);
    setRouteName("");
    setRoutePath("");
    setRouteService("");
    setRouteMethods({ GET: false, POST: false, PUT: false, DELETE: false });
  };

  const handleAddPlugin = () => {
    if (!pluginName.trim()) return;
    const newPlugin: Plugin = {
      name: pluginName,
      scope: pluginScope,
      appliedTo: pluginScope === "global" ? "All routes" : (pluginAppliedTo.trim() || "All routes"),
      config: "custom configuration",
      enabled: true,
    };
    setPlugins((prev) => [...prev, newPlugin]);
    setShowAddPlugin(false);
    setPluginName("jwt-auth");
    setPluginScope("global");
    setPluginAppliedTo("");
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">API Gateway</h1>
          <p className="text-sm text-neutral-500">Kong-powered traffic management and routing</p>
        </div>

        {/* Stats Bar */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Total Requests" value="1.2M/day" />
          <StatCard label="Avg Latency" value="23ms" />
          <StatCard label="Error Rate" value="0.08%" />
          <StatCard label="Active Routes" value={String(routes.filter(r => r.status === "running").length)} />
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
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Route</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Path</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Methods</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Service</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Plugins</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Req/min</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Avg Latency</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {routes.map((route) => (
                  <tr key={route.name} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
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
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{route.service}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {route.plugins.map((plugin) => (
                          <span key={plugin} className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 text-[10px] text-neutral-400">
                            {plugin}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-xs text-neutral-300">{route.reqMin}</td>
                    <td className="px-4 py-3 text-xs text-neutral-300">{route.latency}</td>
                    <td className="px-4 py-3">
                      <StatusBadge status={route.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Plugins */}
        <section>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-white">Plugins</h2>
            <button
              onClick={() => setShowAddPlugin(true)}
              className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
            >
              + Add Plugin
            </button>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Plugin</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Scope</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Applied To</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Config</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Enabled</th>
                </tr>
              </thead>
              <tbody>
                {plugins.map((plugin, i) => (
                  <tr key={`${plugin.name}-${i}`} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{plugin.name}</td>
                    <td className="px-4 py-3">
                      <span className="inline-flex rounded-full bg-surface-300 px-2 py-0.5 text-xs text-neutral-300">
                        {plugin.scope}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-neutral-300">{plugin.appliedTo}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{plugin.config}</td>
                    <td className="px-4 py-3">
                      {plugin.enabled ? (
                        <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
                          <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                          Enabled
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1.5 text-xs text-neutral-500">
                          <span className="h-1.5 w-1.5 rounded-full bg-neutral-500" />
                          Disabled
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Consumers */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Consumers</h2>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Consumer</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Username</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Credentials</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Created</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Requests (24h)</th>
                </tr>
              </thead>
              <tbody>
                {mockConsumers.map((consumer) => (
                  <tr key={consumer.consumer} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{consumer.consumer}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{consumer.username}</td>
                    <td className="px-4 py-3 text-xs text-neutral-300">{consumer.credentials}</td>
                    <td className="px-4 py-3 text-xs text-neutral-500">{consumer.created}</td>
                    <td className="px-4 py-3 text-xs text-neutral-300">{consumer.requests24h}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>

      {showAddRoute && (
        <Modal title="Add Route" onClose={() => setShowAddRoute(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleAddRoute();
            }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input
                type="text"
                value={routeName}
                onChange={(e) => setRouteName(e.target.value)}
                placeholder="my-route"
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
                placeholder="/api/v1/resource/*"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Service</label>
              <input
                type="text"
                value={routeService}
                onChange={(e) => setRouteService(e.target.value)}
                placeholder="service-name:8080"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
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
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                Add Route
              </button>
            </div>
          </form>
        </Modal>
      )}

      {showAddPlugin && (
        <Modal title="Add Plugin" onClose={() => setShowAddPlugin(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleAddPlugin();
            }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <select
                value={pluginName}
                onChange={(e) => setPluginName(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              >
                <option value="jwt-auth">jwt-auth</option>
                <option value="rate-limiting">rate-limiting</option>
                <option value="cors">cors</option>
                <option value="request-transformer">request-transformer</option>
                <option value="ip-restriction">ip-restriction</option>
                <option value="bot-detection">bot-detection</option>
                <option value="hmac-auth">hmac-auth</option>
                <option value="key-auth">key-auth</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Scope</label>
              <select
                value={pluginScope}
                onChange={(e) => setPluginScope(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              >
                <option value="global">global</option>
                <option value="route">route</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Applied To</label>
              <input
                type="text"
                value={pluginAppliedTo}
                onChange={(e) => setPluginAppliedTo(e.target.value)}
                placeholder={pluginScope === "global" ? "All routes" : "route-name"}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => setShowAddPlugin(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
              >
                Add Plugin
              </button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
