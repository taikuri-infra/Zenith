"use client";

import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { Modal } from "@/components/modal";
import { useState } from "react";

interface Planet {
  name: string;
  type: string;
  region: string;
  status: string;
  cpu: string;
  memory: string;
  created: string;
}

const initialPlanets: Planet[] = [
  { name: "planet-01", type: "CX33", region: "fsn1", status: "running", cpu: "4 vCPU", memory: "8 GB", created: "Jan 15, 2026" },
  { name: "planet-02", type: "CX43", region: "fsn1", status: "running", cpu: "8 vCPU", memory: "16 GB", created: "Jan 15, 2026" },
  { name: "planet-03", type: "CPX31", region: "nbg1", status: "running", cpu: "4 vCPU", memory: "8 GB", created: "Jan 22, 2026" },
  { name: "planet-04", type: "CX22", region: "hel1", status: "stopped", cpu: "2 vCPU", memory: "4 GB", created: "Feb 1, 2026" },
  { name: "planet-05", type: "CX33", region: "fsn1", status: "running", cpu: "4 vCPU", memory: "8 GB", created: "Feb 5, 2026" },
];

const typeSpecs: Record<string, { cpu: string; memory: string }> = {
  CX22: { cpu: "2 vCPU", memory: "4 GB" },
  CX33: { cpu: "4 vCPU", memory: "8 GB" },
  CX43: { cpu: "8 vCPU", memory: "16 GB" },
  CPX31: { cpu: "4 vCPU", memory: "8 GB" },
};

export default function PlanetsPage() {
  const [planets, setPlanets] = useState<Planet[]>(initialPlanets);
  const [showAdd, setShowAdd] = useState(false);
  const [formName, setFormName] = useState("");
  const [formType, setFormType] = useState("CX33");
  const [formRegion, setFormRegion] = useState("fsn1");

  const runningCount = planets.filter((p) => p.status === "running").length;

  const handleAdd = () => {
    if (!formName.trim()) return;
    const specs = typeSpecs[formType] || { cpu: "2 vCPU", memory: "4 GB" };
    const newPlanet: Planet = {
      name: formName.trim(),
      type: formType,
      region: formRegion,
      status: "running",
      cpu: specs.cpu,
      memory: specs.memory,
      created: new Date().toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }),
    };
    setPlanets((prev) => [...prev, newPlanet]);
    setShowAdd(false);
    setFormName("");
    setFormType("CX33");
    setFormRegion("fsn1");
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Planets</h1>
            <p className="text-sm text-neutral-500">
              {planets.length} compute nodes, {runningCount} running
            </p>
          </div>
          <button
            onClick={() => setShowAdd(true)}
            className="rounded-lg bg-accent-500 px-3 py-1.5 text-sm text-white hover:bg-accent-600 transition-colors"
          >
            + Add Planet
          </button>
        </div>

        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-100">
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Type</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Region</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Status</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">CPU</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Memory</th>
                <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Created</th>
              </tr>
            </thead>
            <tbody>
              {planets.map((planet) => (
                <tr key={planet.name} className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors">
                  <td className="whitespace-nowrap px-4 py-3 font-medium text-white">{planet.name}</td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <span className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 font-mono text-xs text-neutral-300">{planet.type}</span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">{planet.region}</td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <StatusBadge status={planet.status as "running" | "stopped"} />
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">{planet.cpu}</td>
                  <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">{planet.memory}</td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs text-neutral-400">{planet.created}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {showAdd && (
        <Modal title="Add Planet" onClose={() => setShowAdd(false)}>
          <form onSubmit={(e) => { e.preventDefault(); handleAdd(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Name</label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="planet-06"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Type</label>
              <select
                value={formType}
                onChange={(e) => setFormType(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              >
                <option value="CX22">CX22 (2 vCPU / 4 GB)</option>
                <option value="CX33">CX33 (4 vCPU / 8 GB)</option>
                <option value="CX43">CX43 (8 vCPU / 16 GB)</option>
                <option value="CPX31">CPX31 (4 vCPU / 8 GB)</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Region</label>
              <select
                value={formRegion}
                onChange={(e) => setFormRegion(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              >
                <option value="fsn1">fsn1</option>
                <option value="nbg1">nbg1</option>
                <option value="hel1">hel1</option>
              </select>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowAdd(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Add Planet</button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
