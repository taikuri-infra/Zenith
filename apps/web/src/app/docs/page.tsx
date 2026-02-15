import { Shell } from "@/components/shell";
import { mockApps, mockDatabases, mockPlanets, projectName, projectPlan } from "@/lib/mock-data";

const appConnections: Record<string, string[]> = {
  "frontend": ["api-gateway"],
  "api-gateway": ["user-service", "order-service", "payment-service", "cache"],
  "user-service": ["users-db", "cache"],
  "order-service": ["orders-db", "cache"],
  "payment-service": ["orders-db"],
  "notification-svc": ["users-db"],
};

const connectionStrings: Record<string, string> = {
  "users-db": "postgres://app:********@users-db:5432/users",
  "orders-db": "postgres://app:********@orders-db:5432/orders",
  "cache": "redis://:*****@cache:6379/0",
};

export default function DocsPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Documentation</h1>
          <p className="text-sm text-neutral-500">Auto-generated from your infrastructure</p>
        </div>

        {/* Architecture */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Architecture</h2>
          </div>
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-surface-100">
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">App</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Port</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Replicas</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Connects To</th>
                </tr>
              </thead>
              <tbody>
                {mockApps.map((app) => (
                  <tr key={app.name} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                    <td className="px-4 py-3 font-medium text-white">{app.name}</td>
                    <td className="px-4 py-3 font-mono text-xs text-neutral-400">{app.port}</td>
                    <td className="px-4 py-3 text-neutral-400">
                      {app.replicas.ready}/{app.replicas.total}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {(appConnections[app.name] || []).map((conn) => (
                          <span
                            key={conn}
                            className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 font-mono text-xs text-neutral-300"
                          >
                            {conn}
                          </span>
                        ))}
                        {(!appConnections[app.name] || appConnections[app.name].length === 0) && (
                          <span className="text-xs text-neutral-600">--</span>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Databases */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Databases</h2>
          </div>
          <div className="space-y-2">
            {mockDatabases.map((db) => (
              <div key={db.name} className="rounded-lg border border-border bg-surface-100 p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium text-white">{db.name}</p>
                    <span className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 text-xs text-neutral-400 capitalize">
                      {db.engine} {db.version}
                    </span>
                  </div>
                  <span className="text-xs text-neutral-500">
                    {db.storageUsed} / {db.storageTotal}
                  </span>
                </div>
                {connectionStrings[db.name] && (
                  <div className="mt-2">
                    <p className="mb-1 text-xs text-neutral-500">Connection String</p>
                    <div className="rounded bg-surface-200 px-3 py-2">
                      <code className="font-mono text-xs text-neutral-400">{connectionStrings[db.name]}</code>
                    </div>
                  </div>
                )}
                <div className="mt-2">
                  <p className="mb-1 text-xs text-neutral-500">Linked Apps</p>
                  <div className="flex flex-wrap gap-1">
                    {db.linkedApps.map((app) => (
                      <span
                        key={app}
                        className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 font-mono text-xs text-neutral-300"
                      >
                        {app}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* Environment */}
        <section>
          <div className="mb-3">
            <h2 className="text-sm font-medium text-white">Environment</h2>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-5">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-xs text-neutral-500">Project</p>
                <p className="mt-0.5 text-sm font-medium text-white">{projectName}</p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Plan</p>
                <p className="mt-0.5 text-sm font-medium text-white">{projectPlan}</p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Apps</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {mockApps.length} services ({mockApps.filter((a) => a.status === "running").length} running)
                </p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Databases</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {mockDatabases.length} instances ({mockDatabases.filter((d) => d.status === "running").length} running)
                </p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Compute</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {mockPlanets.length} planets &middot; {mockPlanets.reduce((s, p) => s + p.cpuCores, 0)} vCPU &middot; {mockPlanets.reduce((s, p) => s + p.ramGb, 0)}GB RAM
                </p>
              </div>
              <div>
                <p className="text-xs text-neutral-500">Region</p>
                <p className="mt-0.5 text-sm font-medium text-white">
                  {[...new Set(mockPlanets.map((p) => p.region))].join(", ")}
                </p>
              </div>
            </div>
          </div>
        </section>
      </div>
    </Shell>
  );
}
