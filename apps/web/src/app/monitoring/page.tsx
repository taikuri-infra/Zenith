import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";

const dashboards = [
  { name: "Platform Overview", type: "overview" as const, panels: 12, lastViewed: "2 hours ago" },
  { name: "Service Health", type: "service" as const, panels: 8, lastViewed: "35 min ago" },
  { name: "Node Metrics", type: "infrastructure" as const, panels: 6, lastViewed: "1 day ago" },
  { name: "API Latency Analysis", type: "custom" as const, panels: 4, lastViewed: "5 hours ago" },
];

const dashboardTypeBadge: Record<string, { bg: string; text: string }> = {
  overview: { bg: "bg-accent-500/15", text: "text-accent-400" },
  service: { bg: "bg-emerald-500/15", text: "text-emerald-400" },
  infrastructure: { bg: "bg-amber-500/15", text: "text-amber-400" },
  custom: { bg: "bg-purple-500/15", text: "text-purple-400" },
};

const prometheusTargets = [
  { target: "api-gateway", endpoint: "http://api-gateway:9090/metrics", status: "up", lastScrape: "15s ago", scrapeDuration: "12ms", samples: "1,247" },
  { target: "user-service", endpoint: "http://user-service:9090/metrics", status: "up", lastScrape: "15s ago", scrapeDuration: "8ms", samples: "892" },
  { target: "order-service", endpoint: "http://order-service:9090/metrics", status: "up", lastScrape: "15s ago", scrapeDuration: "11ms", samples: "743" },
  { target: "payment-service", endpoint: "http://payment-service:9090/metrics", status: "up", lastScrape: "15s ago", scrapeDuration: "6ms", samples: "421" },
  { target: "node-exporter", endpoint: "http://planet-01:9100/metrics", status: "up", lastScrape: "30s ago", scrapeDuration: "23ms", samples: "3,891" },
  { target: "postgres-exporter", endpoint: "http://users-db:9187/metrics", status: "up", lastScrape: "30s ago", scrapeDuration: "45ms", samples: "567" },
];

const mockLogs = [
  { time: "14:23:01.432", level: "INF", service: "api-gateway", message: "\u2192 POST /api/v1/orders 201 89ms consumer=web-app" },
  { time: "14:23:01.510", level: "INF", service: "user-service", message: "cache hit uid=8a3f2 latency=0.3ms" },
  { time: "14:23:02.101", level: "WRN", service: "order-service", message: "slow query: SELECT * FROM orders WHERE... (342ms)" },
  { time: "14:23:02.450", level: "DBG", service: "payment-service", message: "stripe webhook received event=payment_intent.succeeded" },
  { time: "14:23:02.892", level: "INF", service: "frontend", message: "SSR render /dashboard 45ms cache=HIT" },
  { time: "14:23:03.220", level: "ERR", service: "payment-service", message: "stripe webhook sig verification failed req=w9x2k" },
  { time: "14:23:03.567", level: "INF", service: "api-gateway", message: "\u2192 GET /api/v1/users/me 200 12ms consumer=mobile-app" },
  { time: "14:23:03.891", level: "INF", service: "order-service", message: "order ord_29f1k created total=\u20ac49.99" },
  { time: "14:23:04.102", level: "WRN", service: "api-gateway", message: "rate limit 85/100 rpm for consumer=partner-api" },
  { time: "14:23:04.334", level: "INF", service: "user-service", message: "jwt refreshed uid=3b7e1 exp=+1h" },
  { time: "14:23:04.567", level: "ERR", service: "notification-svc", message: "SMTP connection timeout after 30s host=smtp.eu.mailgun.org" },
  { time: "14:23:05.001", level: "INF", service: "auth-service", message: "token issued realm=production client=web-app sub=sarah@startup.com" },
];

const levelColors: Record<string, string> = {
  INF: "text-emerald-400",
  WRN: "text-amber-400",
  ERR: "text-red-400",
  DBG: "text-neutral-500",
};

export default function MonitoringPage() {
  return (
    <Shell>
      <div className="space-y-8">
        <div>
          <h1 className="text-lg font-semibold text-white">Monitoring</h1>
          <p className="text-sm text-neutral-500">Performance metrics, dashboards, and log aggregation</p>
        </div>

        {/* Stats Row */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Avg Response" value="23ms" sub="p50" />
          <StatCard label="P99 Latency" value="142ms" sub="across all services" />
          <StatCard label="Error Rate" value="0.08%" sub="last 24h" />
          <StatCard label="Uptime" value="99.97%" sub="30 day SLA" />
        </div>

        {/* Grafana Dashboards */}
        <section>
          <div className="mb-4">
            <h2 className="text-sm font-medium text-white">Dashboards</h2>
            <p className="mt-0.5 text-xs text-neutral-500">Grafana-powered metrics visualization</p>
          </div>
          <div className="grid grid-cols-2 gap-4">
            {dashboards.map((d) => {
              const badge = dashboardTypeBadge[d.type];
              return (
                <div
                  key={d.name}
                  className="cursor-pointer rounded-lg border border-border bg-surface-100 p-4 transition-colors hover:border-border-hover hover:bg-surface-200"
                >
                  <div className="mb-3 flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-white">{d.name}</p>
                      <div className="mt-1.5 flex items-center gap-3">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-[10px] font-medium ${badge.bg} ${badge.text}`}>
                          {d.type}
                        </span>
                        <span className="text-xs text-neutral-500">{d.panels} panels</span>
                        <span className="text-xs text-neutral-600">Last viewed {d.lastViewed}</span>
                      </div>
                    </div>
                  </div>
                  <div className="flex h-24 items-center justify-center rounded bg-surface-200">
                    <svg className="h-6 w-6 text-neutral-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
                    </svg>
                  </div>
                </div>
              );
            })}
          </div>
        </section>

        {/* Prometheus Metrics */}
        <section>
          <div className="mb-4">
            <h2 className="text-sm font-medium text-white">Metrics</h2>
            <p className="mt-0.5 text-xs text-neutral-500">Prometheus targets</p>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Target</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Endpoint</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Status</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Scrape</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Scrape Duration</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Samples</th>
                </tr>
              </thead>
              <tbody>
                {prometheusTargets.map((t) => (
                  <tr key={t.target} className="border-t border-border transition-colors hover:bg-surface-200">
                    <td className="px-4 py-2.5 text-sm font-medium text-white">{t.target}</td>
                    <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">{t.endpoint}</td>
                    <td className="px-4 py-2.5">
                      <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
                        <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                        up
                      </span>
                    </td>
                    <td className="px-4 py-2.5 font-mono text-xs text-neutral-500">{t.lastScrape}</td>
                    <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">{t.scrapeDuration}</td>
                    <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">{t.samples}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Loki Logs */}
        <section>
          <div className="mb-4">
            <h2 className="text-sm font-medium text-white">Logs</h2>
            <p className="mt-0.5 text-xs text-neutral-500">Loki log aggregation</p>
          </div>

          {/* Filter bar */}
          <div className="mb-3 flex items-center gap-3">
            <select className="rounded-md border border-border bg-surface-200 px-3 py-1.5 text-xs text-neutral-300 outline-none">
              <option>All Services</option>
              <option>api-gateway</option>
              <option>user-service</option>
              <option>order-service</option>
              <option>payment-service</option>
              <option>frontend</option>
              <option>notification-svc</option>
              <option>auth-service</option>
            </select>
            <select className="rounded-md border border-border bg-surface-200 px-3 py-1.5 text-xs text-neutral-300 outline-none">
              <option>All Levels</option>
              <option>INF</option>
              <option>WRN</option>
              <option>ERR</option>
              <option>DBG</option>
            </select>
            <div className="relative flex-1">
              <svg className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <input
                type="text"
                placeholder="Search logs..."
                className="w-full rounded-md border border-border bg-surface-200 py-1.5 pl-8 pr-3 text-xs text-neutral-300 outline-none placeholder:text-neutral-600 focus:border-border-hover"
              />
            </div>
          </div>

          {/* Log output */}
          <div className="overflow-x-auto rounded-lg bg-[#0d1117] p-4">
            <div className="space-y-0.5">
              {mockLogs.map((log, i) => (
                <div key={i} className="flex gap-2 font-mono text-xs leading-5">
                  <span className="flex-shrink-0 text-neutral-600">{log.time}</span>
                  <span className={`w-7 flex-shrink-0 font-semibold ${levelColors[log.level]}`}>{log.level}</span>
                  <span className="flex-shrink-0 text-accent-400">[{log.service}]</span>
                  <span className="text-neutral-400">{log.message}</span>
                </div>
              ))}
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
