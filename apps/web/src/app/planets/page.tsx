import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { ProgressBar } from "@/components/progress-bar";
import { mockPlanets } from "@/lib/mock-data";

export default function PlanetsPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Planets</h1>
            <p className="text-sm text-neutral-500">Compute nodes powering your project</p>
          </div>
          <button className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors">
            + Add Planet
          </button>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {mockPlanets.map((planet) => (
            <div
              key={planet.name}
              className="rounded-lg border border-border bg-surface-100 p-4 transition-colors hover:border-border-hover"
            >
              {/* Header */}
              <div className="mb-4 flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-white">{planet.name}</p>
                  <p className="text-xs text-neutral-500">{planet.region}</p>
                </div>
                <div className="flex items-center gap-2">
                  <span className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 font-mono text-xs text-neutral-300">
                    {planet.size}
                  </span>
                  <StatusBadge status={planet.status} />
                </div>
              </div>

              {/* CPU */}
              <div className="mb-3">
                <div className="mb-1 flex items-center justify-between">
                  <span className="text-xs text-neutral-500">CPU</span>
                  <span className="text-xs tabular-nums text-neutral-400">{planet.cpuCores} vCPU</span>
                </div>
                <ProgressBar percent={planet.cpuPercent} label={`${planet.cpuPercent}%`} />
              </div>

              {/* RAM */}
              <div>
                <div className="mb-1 flex items-center justify-between">
                  <span className="text-xs text-neutral-500">RAM</span>
                  <span className="text-xs tabular-nums text-neutral-400">{planet.ramGb}GB RAM</span>
                </div>
                <ProgressBar percent={planet.ramPercent} label={`${planet.ramPercent}%`} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </Shell>
  );
}
