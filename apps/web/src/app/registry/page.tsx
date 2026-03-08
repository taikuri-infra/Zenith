"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { Modal } from "@/components/modal";
import { useState, useEffect } from "react";
import { getApi } from "@/lib/get-api";
import type { RegistryRepo, RegistryArtifact } from "@/lib/api";

export default function RegistryPage() {
  const [copied, setCopied] = useState(false);
  const [repos, setRepos] = useState<RegistryRepo[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [repoName, setRepoName] = useState("");
  const [repoVisibility, setRepoVisibility] = useState("private");

  useEffect(() => {
    const api = getApi();
    api.registry.listRepos().then((data) => {
      setRepos(data);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  const pullCommand = repos.length > 0
    ? `docker pull registry.freezenith.com/${repos[0].name}:latest`
    : "docker pull registry.freezenith.com/your-app:latest";

  const handleCopy = () => {
    navigator.clipboard.writeText(pullCommand);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleCreate = () => {
    if (!repoName.trim()) return;
    const newRepo: RegistryRepo = {
      name: repoName.trim(),
      artifact_count: 0,
      last_pushed: "never",
      artifacts: [],
      scan: { passed: 0, warning: 0, failed: 0, total: 0 },
    };
    setRepos((prev) => [...prev, newRepo]);
    setShowCreate(false);
    setRepoName("");
    setRepoVisibility("private");
  };

  const totalTags = repos.reduce((sum, r) => sum + r.artifact_count, 0);
  const totalPassed = repos.reduce((sum, r) => sum + (r.scan?.passed ?? 0), 0);
  const totalWarnings = repos.reduce((sum, r) => sum + (r.scan?.warning ?? 0), 0);
  const totalFailed = repos.reduce((sum, r) => sum + (r.scan?.failed ?? 0), 0);

  function scanStatusLabel(scan?: RegistryRepo["scan"]) {
    if (!scan || scan.total === 0) return <span className="text-xs text-neutral-500">No images</span>;
    if (scan.failed > 0) {
      return <span className="text-xs text-red-400">{scan.passed}/{scan.total} passed, {scan.failed} failed</span>;
    }
    if (scan.warning > 0) {
      return <span className="text-xs text-amber-400">{scan.passed}/{scan.total} passed, {scan.warning} warning</span>;
    }
    return <span className="text-xs text-emerald-400">{scan.passed}/{scan.total} passed</span>;
  }

  function artifactStatusIcon(status: string) {
    if (status === "passed") {
      return (
        <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
          passed
        </span>
      );
    }
    if (status === "failed") {
      return (
        <span className="inline-flex items-center gap-1.5 text-xs text-red-400">
          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
          failed
        </span>
      );
    }
    if (status === "warning") {
      return (
        <span className="inline-flex items-center gap-1.5 text-xs text-amber-400">
          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
          warning
        </span>
      );
    }
    return <span className="inline-flex items-center gap-1.5 text-xs text-neutral-500">pending</span>;
  }

  return (
    <Shell>
      <div className="space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Container Registry</h1>
            <p className="text-sm text-neutral-500">Private container image repository with vulnerability scanning</p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-2 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-600"
          >
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create Repository
          </button>
        </div>

        {/* Stats Row */}
        <div className="grid grid-cols-4 gap-4">
          <StatCard label="Repositories" value={loading ? "..." : String(repos.length)} />
          <StatCard label="Total Images" value={loading ? "..." : `${totalTags} tags`} />
          <StatCard label="Scans Passed" value={loading ? "..." : String(totalPassed)} sub={totalFailed > 0 ? `${totalFailed} failed` : undefined} />
          <StatCard label="Warnings" value={loading ? "..." : String(totalWarnings)} sub={totalFailed > 0 ? `${totalFailed} critical/high` : undefined} />
        </div>

        {/* Loading skeleton */}
        {loading && (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-16 animate-pulse rounded-lg border border-border bg-surface-100" />
            ))}
          </div>
        )}

        {/* Repositories Table */}
        {!loading && repos.length > 0 && (
          <section>
            <div className="mb-4">
              <h2 className="text-sm font-medium text-white">Repositories</h2>
            </div>
            <div className="overflow-hidden rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="bg-surface-100">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Repository</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Tags</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Pushed</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Scan Status</th>
                  </tr>
                </thead>
                <tbody>
                  {repos.map((repo) => (
                    <tr key={repo.name} className="border-t border-border transition-colors hover:bg-surface-200">
                      <td className="px-4 py-2.5">
                        <div className="flex items-center gap-2.5">
                          <div className="flex h-7 w-7 items-center justify-center rounded bg-accent-500/10">
                            <svg className="h-3.5 w-3.5 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                            </svg>
                          </div>
                          <span className="text-sm font-medium text-white">{repo.name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-2.5 text-xs text-neutral-400">{repo.artifact_count}</td>
                      <td className="px-4 py-2.5 text-xs text-neutral-500">{repo.last_pushed || "never"}</td>
                      <td className="px-4 py-2.5">{scanStatusLabel(repo.scan)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        )}

        {/* Image Details with Scan Results */}
        {!loading && repos.length > 0 && (
          <section>
            <div className="mb-4">
              <h2 className="text-sm font-medium text-white">Image Details &amp; Vulnerability Scan</h2>
              <p className="mt-0.5 text-xs text-neutral-500">Expand a repository to view individual image tags and Trivy scan results</p>
            </div>
            <div className="space-y-3">
              {repos.map((repo) => (
                <details key={repo.name} className="group overflow-hidden rounded-lg border border-border bg-surface-100">
                  <summary className="flex cursor-pointer items-center justify-between px-5 py-3.5 transition-colors hover:bg-surface-200">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent-500/10">
                        <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                        </svg>
                      </div>
                      <div>
                        <p className="text-sm font-medium text-white">{repo.name}</p>
                        <p className="text-xs text-neutral-500">
                          {repo.artifact_count} tag{repo.artifact_count !== 1 ? "s" : ""}
                        </p>
                      </div>
                    </div>
                    <svg
                      className="h-4 w-4 text-neutral-500 transition-transform group-open:rotate-180"
                      fill="none" viewBox="0 0 24 24" stroke="currentColor"
                    >
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                    </svg>
                  </summary>
                  <div className="border-t border-border">
                    {repo.artifacts && repo.artifacts.length > 0 ? (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="bg-surface-100">
                            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Tag</th>
                            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Digest</th>
                            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Size</th>
                            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Pushed</th>
                            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Scan Status</th>
                            <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Vulnerabilities</th>
                          </tr>
                        </thead>
                        <tbody>
                          {repo.artifacts.map((img: RegistryArtifact) => (
                            <tr key={img.tag + img.digest} className="border-t border-border transition-colors hover:bg-surface-200">
                              <td className="px-4 py-2.5">
                                <span className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 font-mono text-xs text-neutral-300">
                                  {img.tag}
                                </span>
                              </td>
                              <td className="px-4 py-2.5 font-mono text-xs text-neutral-500">{img.digest.substring(0, 16)}</td>
                              <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">{img.size}</td>
                              <td className="px-4 py-2.5 text-xs text-neutral-500">{img.pushed}</td>
                              <td className="px-4 py-2.5">{artifactStatusIcon(img.status)}</td>
                              <td className="px-4 py-2.5">
                                <div className="flex items-center gap-3 text-xs">
                                  <span className={img.critical > 0 ? "text-red-400" : "text-neutral-600"}>
                                    {img.critical} critical
                                  </span>
                                  <span className={img.high > 0 ? "text-amber-400" : "text-neutral-600"}>
                                    {img.high} high
                                  </span>
                                  {img.medium > 0 && (
                                    <span className="text-neutral-500">{img.medium} medium</span>
                                  )}
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    ) : (
                      <div className="px-5 py-4 text-center text-xs text-neutral-500">No images yet. Push your first image to this repository.</div>
                    )}
                  </div>
                </details>
              ))}
            </div>
          </section>
        )}

        {/* Empty state */}
        {!loading && repos.length === 0 && (
          <div className="rounded-xl border border-border bg-surface-100 py-16 text-center">
            <svg className="mx-auto mb-3 h-10 w-10 text-neutral-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            <p className="text-sm text-neutral-400">No repositories yet</p>
            <p className="text-xs text-neutral-500 mt-1">Push your first image to get started</p>
          </div>
        )}

        {/* Image Pull Commands */}
        {!loading && repos.length > 0 && (
          <section>
            <div className="mb-4">
              <h2 className="text-sm font-medium text-white">Image Pull Commands</h2>
            </div>
            <div className="flex items-center justify-between rounded-lg border border-border bg-[#0d1117] px-4 py-3">
              <code className="font-mono text-sm text-neutral-300">{pullCommand}</code>
              <button
                onClick={handleCopy}
                className="ml-4 flex-shrink-0 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-neutral-400 transition-colors hover:border-border-hover hover:text-neutral-300"
              >
                {copied ? "Copied!" : "Copy"}
              </button>
            </div>
          </section>
        )}
      </div>

      {showCreate && (
        <Modal title="Create Repository" onClose={() => setShowCreate(false)}>
          <form onSubmit={(e) => { e.preventDefault(); handleCreate(); }} className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Repository Name</label>
              <input
                type="text"
                value={repoName}
                onChange={(e) => setRepoName(e.target.value)}
                placeholder="my-service"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Visibility</label>
              <select
                value={repoVisibility}
                onChange={(e) => setRepoVisibility(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              >
                <option value="private">private</option>
                <option value="public">public</option>
              </select>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button type="button" onClick={() => setShowCreate(false)} className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors">Cancel</button>
              <button type="submit" className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors">Create</button>
            </div>
          </form>
        </Modal>
      )}
    </Shell>
  );
}
