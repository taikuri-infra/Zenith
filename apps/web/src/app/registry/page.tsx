"use client";

import { Shell } from "@/components/shell";
import { StatCard } from "@/components/stat-card";
import { Modal } from "@/components/modal";
import { useState } from "react";

interface Repository {
  name: string;
  tags: number;
  totalSize: string;
  lastPushed: string;
  scan: { passed: number; total: number; warning: number };
  lifecycle: string;
  uri: string;
  visibility: string;
  images: {
    tag: string;
    digest: string;
    size: string;
    pushed: string;
    status: "passed" | "warning";
    critical: number;
    high: number;
    medium: number;
  }[];
}

const initialRepositories: Repository[] = [
  {
    name: "frontend",
    tags: 3,
    totalSize: "423MB",
    lastPushed: "2 hours ago",
    scan: { passed: 3, total: 3, warning: 0 },
    lifecycle: "Keep last 10",
    uri: "registry.zenith.cloud/my-startup/frontend",
    visibility: "private",
    images: [
      { tag: "latest", digest: "sha256:a1b2c3d4", size: "142MB", pushed: "2 hours ago", status: "passed", critical: 0, high: 0, medium: 0 },
      { tag: "v1.4.2", digest: "sha256:a1b2c3d4", size: "142MB", pushed: "2 hours ago", status: "passed", critical: 0, high: 0, medium: 0 },
      { tag: "v1.4.1", digest: "sha256:e5f6g7h8", size: "139MB", pushed: "3 days ago", status: "passed", critical: 0, high: 0, medium: 2 },
    ],
  },
  {
    name: "api-gateway",
    tags: 2,
    totalSize: "196MB",
    lastPushed: "5 hours ago",
    scan: { passed: 2, total: 2, warning: 0 },
    lifecycle: "Keep last 10",
    uri: "registry.zenith.cloud/my-startup/api-gateway",
    visibility: "private",
    images: [
      { tag: "latest", digest: "sha256:i9j0k1l2", size: "98MB", pushed: "5 hours ago", status: "passed", critical: 0, high: 0, medium: 0 },
      { tag: "v2.1.0", digest: "sha256:i9j0k1l2", size: "98MB", pushed: "5 hours ago", status: "passed", critical: 0, high: 0, medium: 0 },
    ],
  },
  {
    name: "user-service",
    tags: 4,
    totalSize: "442MB",
    lastPushed: "1 day ago",
    scan: { passed: 3, total: 4, warning: 1 },
    lifecycle: "Keep last 5",
    uri: "registry.zenith.cloud/my-startup/user-service",
    visibility: "private",
    images: [
      { tag: "latest", digest: "sha256:m3n4o5p6", size: "112MB", pushed: "1 day ago", status: "warning", critical: 0, high: 1, medium: 0 },
      { tag: "v3.0.1", digest: "sha256:m3n4o5p6", size: "112MB", pushed: "1 day ago", status: "warning", critical: 0, high: 1, medium: 0 },
      { tag: "v3.0.0", digest: "sha256:q7r8s9t0", size: "110MB", pushed: "4 days ago", status: "passed", critical: 0, high: 0, medium: 0 },
      { tag: "v2.9.8", digest: "sha256:u1v2w3x4", size: "108MB", pushed: "1 week ago", status: "passed", critical: 0, high: 0, medium: 0 },
    ],
  },
];

export default function RegistryPage() {
  const [copied, setCopied] = useState(false);
  const [repositories, setRepositories] = useState<Repository[]>(initialRepositories);
  const [showCreate, setShowCreate] = useState(false);
  const [repoName, setRepoName] = useState("");
  const [repoVisibility, setRepoVisibility] = useState("private");

  const pullCommand = "docker pull registry.zenith.cloud/my-startup/frontend:latest";

  const handleCopy = () => {
    navigator.clipboard.writeText(pullCommand);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleCreate = () => {
    if (!repoName.trim()) return;
    const newRepo: Repository = {
      name: repoName.trim(),
      tags: 0,
      totalSize: "0 B",
      lastPushed: "never",
      scan: { passed: 0, total: 0, warning: 0 },
      lifecycle: "Keep last 10",
      uri: `registry.zenith.cloud/my-startup/${repoName.trim()}`,
      visibility: repoVisibility,
      images: [],
    };
    setRepositories((prev) => [...prev, newRepo]);
    setShowCreate(false);
    setRepoName("");
    setRepoVisibility("private");
  };

  const totalTags = repositories.reduce((sum, r) => sum + r.tags, 0);
  const totalWarnings = repositories.reduce((sum, r) => sum + r.scan.warning, 0);
  const totalPassed = repositories.reduce((sum, r) => sum + r.scan.passed, 0);

  return (
    <Shell>
      <div className="space-y-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Container Registry</h1>
            <p className="text-sm text-neutral-500">Private container image repository</p>
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
          <StatCard label="Repositories" value={String(repositories.length)} />
          <StatCard label="Total Images" value={`${totalTags} tags`} />
          <StatCard label="Total Size" value="1.06 GB" />
          <StatCard label="Scan Status" value={`${totalPassed} passed`} sub={totalWarnings > 0 ? `${totalWarnings} warning` : undefined} />
        </div>

        {/* Repositories Table */}
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
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Size</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Last Pushed</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Scan</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">Lifecycle Policy</th>
                  <th className="px-4 py-2.5 text-left text-xs font-medium text-neutral-500">URI</th>
                </tr>
              </thead>
              <tbody>
                {repositories.map((repo) => (
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
                    <td className="px-4 py-2.5 text-xs text-neutral-400">{repo.tags}</td>
                    <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">{repo.totalSize}</td>
                    <td className="px-4 py-2.5 text-xs text-neutral-500">{repo.lastPushed}</td>
                    <td className="px-4 py-2.5">
                      {repo.scan.warning > 0 ? (
                        <span className="text-xs text-amber-400">
                          {repo.scan.passed}/{repo.scan.total} passed, {repo.scan.warning} warning
                        </span>
                      ) : repo.scan.total > 0 ? (
                        <span className="text-xs text-emerald-400">
                          {repo.scan.passed}/{repo.scan.total} passed
                        </span>
                      ) : (
                        <span className="text-xs text-neutral-500">No images</span>
                      )}
                    </td>
                    <td className="px-4 py-2.5 text-xs text-neutral-500">{repo.lifecycle}</td>
                    <td className="max-w-[220px] truncate px-4 py-2.5 font-mono text-xs text-neutral-500">{repo.uri}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Image Details */}
        <section>
          <div className="mb-4">
            <h2 className="text-sm font-medium text-white">Image Details</h2>
            <p className="mt-0.5 text-xs text-neutral-500">Expand a repository to view individual image tags</p>
          </div>
          <div className="space-y-3">
            {repositories.map((repo) => (
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
                        {repo.tags} tag{repo.tags !== 1 ? "s" : ""} &middot; {repo.totalSize}
                      </p>
                    </div>
                  </div>
                  <svg
                    className="h-4 w-4 text-neutral-500 transition-transform group-open:rotate-180"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </summary>
                <div className="border-t border-border">
                  {repo.images.length > 0 ? (
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
                        {repo.images.map((img) => (
                          <tr key={img.tag} className="border-t border-border transition-colors hover:bg-surface-200">
                            <td className="px-4 py-2.5">
                              <span className="inline-flex rounded bg-surface-300 px-1.5 py-0.5 font-mono text-xs text-neutral-300">
                                {img.tag}
                              </span>
                            </td>
                            <td className="px-4 py-2.5 font-mono text-xs text-neutral-500">{img.digest}</td>
                            <td className="px-4 py-2.5 font-mono text-xs text-neutral-400">{img.size}</td>
                            <td className="px-4 py-2.5 text-xs text-neutral-500">{img.pushed}</td>
                            <td className="px-4 py-2.5">
                              {img.status === "passed" ? (
                                <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
                                  <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                  </svg>
                                  passed
                                </span>
                              ) : (
                                <span className="inline-flex items-center gap-1.5 text-xs text-amber-400">
                                  <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
                                  </svg>
                                  warning
                                </span>
                              )}
                            </td>
                            <td className="px-4 py-2.5">
                              <div className="flex items-center gap-3 text-xs">
                                <span className={img.critical > 0 ? "text-red-400" : "text-neutral-600"}>
                                  {img.critical} critical
                                </span>
                                <span className={img.high > 0 ? "text-amber-400" : "text-neutral-600"}>
                                  {img.high} high
                                </span>
                                {img.medium > 0 && (
                                  <span className="text-neutral-500">
                                    {img.medium} medium
                                  </span>
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

        {/* Image Pull Commands */}
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
