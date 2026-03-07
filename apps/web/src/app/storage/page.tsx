"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { useProject } from "@/hooks/use-project";
import { type StorageBucketV2 } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import Link from "next/link";
import { useState, useMemo } from "react";

function formatBytes(mb: number): string {
  if (mb === 0) return "0 B";
  if (mb < 1) return `${(mb * 1024).toFixed(0)} KB`;
  if (mb < 1024) return `${mb.toFixed(1)} MB`;
  return `${(mb / 1024).toFixed(2)} GB`;
}

export default function StoragePage() {
  const { storageBuckets, userPlan } = getApi();
  const projectId = useProject();

  const {
    data: bucketList,
    loading,
    error,
    refetch,
  } = useApi(() => storageBuckets.list(projectId || undefined), [projectId]);

  const { data: planData, loading: planLoading } = useApi(
    () => userPlan.get(),
    []
  );

  const [showCreate, setShowCreate] = useState(false);
  const [formName, setFormName] = useState("");
  const [formAccess, setFormAccess] = useState("private");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [search, setSearch] = useState("");
  const [accessFilter, setAccessFilter] = useState("");

  const buckets: StorageBucketV2[] = bucketList ?? [];
  const maxBuckets = planData?.limits?.max_buckets ?? 0;
  const tier = planData?.tier ?? "free";
  const isFree = tier === "free";

  const filtered = useMemo(() => {
    let result = buckets;
    if (search) {
      const q = search.toLowerCase();
      result = result.filter((b) => b.name.toLowerCase().includes(q));
    }
    if (accessFilter) {
      result = result.filter((b) => b.access === accessFilter);
    }
    return result;
  }, [buckets, search, accessFilter]);

  const totalObjects = buckets.reduce((sum, b) => sum + b.objects, 0);
  const totalSize = buckets.reduce((sum, b) => sum + b.size_mb, 0);

  const handleCreate = async () => {
    if (!formName.trim() || creating) return;
    setCreating(true);
    setCreateError("");
    try {
      await storageBuckets.create({ name: formName.trim(), access: formAccess });
      setShowCreate(false);
      setFormName("");
      setFormAccess("private");
      refetch();
    } catch (err: unknown) {
      const status = (err as { status?: number }).status;
      if (status === 403) {
        setCreateError(
          "You've reached your plan's bucket limit. Upgrade to Pro for up to 5 buckets."
        );
      } else {
        setCreateError(
          err instanceof Error ? err.message : "Failed to create bucket"
        );
      }
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteId || deleting) return;
    setDeleting(true);
    try {
      await storageBuckets.delete(deleteId);
      setDeleteId(null);
      refetch();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete bucket");
    } finally {
      setDeleting(false);
    }
  };

  if (loading || planLoading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={7} rows={3} />
      </Shell>
    );
  }

  if (error) {
    return (
      <Shell>
        <ErrorState message={error} onRetry={refetch} />
      </Shell>
    );
  }

  // Free tier with no buckets — show full upgrade prompt
  if (isFree && maxBuckets === 0 && buckets.length === 0) {
    return (
      <Shell>
        <div className="space-y-6">
          <div>
            <h1 className="text-lg font-semibold text-white">Storage</h1>
            <p className="text-sm text-neutral-500">
              S3-compatible object storage buckets
            </p>
          </div>

          <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-surface-100 py-16 px-6">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-accent-500/10 mb-5">
              <svg
                className="h-8 w-8 text-accent-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={1.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75"
                />
              </svg>
            </div>
            <h2 className="text-xl font-semibold text-white mb-2">
              Object Storage is a Pro Feature
            </h2>
            <p className="text-sm text-neutral-400 text-center max-w-md mb-6">
              Store files, images, and assets with S3-compatible object storage.
              Upload, download, and organize objects in private or public buckets.
            </p>
            <div className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-xs text-neutral-500 mb-8">
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Up to 5 buckets
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                10 GB storage
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                File browser
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Public &amp; private access
              </span>
            </div>
            <Link
              href="/billing"
              className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-6 py-2.5 text-sm font-medium transition-colors"
            >
              Upgrade to Pro
            </Link>
          </div>
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Storage</h1>
            <p className="text-sm text-neutral-500">
              S3-compatible object storage buckets
            </p>
          </div>
          <div className="flex items-center gap-3">
            {maxBuckets > 0 && (
              <span className="text-xs text-neutral-500">
                {buckets.length} / {maxBuckets} buckets
              </span>
            )}
            <button
              onClick={() => {
                setCreateError("");
                setShowCreate(true);
              }}
              className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors"
            >
              + Create Bucket
            </button>
          </div>
        </div>

        {/* Stat cards */}
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">
              Total Buckets
            </p>
            <p className="mt-1 text-2xl font-semibold tabular-nums text-white">
              {buckets.length}
            </p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">
              Total Objects
            </p>
            <p className="mt-1 text-2xl font-semibold tabular-nums text-white">
              {totalObjects.toLocaleString()}
            </p>
          </div>
          <div className="rounded-lg border border-border bg-surface-100 p-4">
            <p className="text-xs font-medium text-neutral-500">Total Size</p>
            <p className="mt-1 text-2xl font-semibold tabular-nums text-white">
              {formatBytes(totalSize)}
            </p>
          </div>
        </div>

        {/* Filter bar */}
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <svg
              className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M21 21l-4.35-4.35M11 19a8 8 0 100-16 8 8 0 000 16z"
              />
            </svg>
            <input
              type="text"
              placeholder="Filter buckets..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-border bg-surface-100 py-1.5 pl-9 pr-3 text-sm text-white placeholder:text-neutral-500 focus:border-accent-500 focus:outline-none"
            />
          </div>
          <select
            value={accessFilter}
            onChange={(e) => setAccessFilter(e.target.value)}
            className="rounded-lg border border-border bg-surface-100 px-3 py-1.5 text-sm text-neutral-400 focus:border-accent-500 focus:outline-none"
          >
            <option value="">All access</option>
            <option value="private">Private</option>
            <option value="public">Public</option>
          </select>
        </div>

        {/* Table or Empty State */}
        {filtered.length === 0 ? (
          <EmptyState
            title="No storage buckets yet"
            description="Create a storage bucket to store files and objects."
            actionLabel="+ Create Bucket"
          />
        ) : (
          <div className="overflow-hidden rounded-lg border border-border">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border bg-surface-100">
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Bucket Name
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Objects
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Size
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Access
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Status
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Region
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">
                      Created
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500"></th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((bucket) => (
                    <tr
                      key={bucket.id}
                      className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                    >
                      <td className="whitespace-nowrap px-4 py-3">
                        <Link
                          href={`/storage/${bucket.id}`}
                          className="font-medium text-white hover:text-accent-400 transition-colors"
                        >
                          {bucket.name}
                        </Link>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs tabular-nums text-neutral-300">
                        {bucket.objects.toLocaleString()}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-300">
                        {formatBytes(bucket.size_mb)}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3">
                        {bucket.access === "private" ? (
                          <span className="inline-flex items-center rounded-full bg-neutral-500/10 px-2 py-0.5 text-xs font-medium text-neutral-400">
                            Private
                          </span>
                        ) : (
                          <span className="inline-flex items-center rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400">
                            Public
                          </span>
                        )}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3">
                        <span
                          className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium capitalize ${
                            bucket.status === "active"
                              ? "bg-emerald-500/10 text-emerald-400"
                              : "bg-amber-500/10 text-amber-400"
                          }`}
                        >
                          <span
                            className={`h-1.5 w-1.5 rounded-full ${
                              bucket.status === "active"
                                ? "bg-emerald-400"
                                : "bg-amber-400"
                            }`}
                          />
                          {bucket.status}
                        </span>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                        {bucket.region}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                        {new Date(bucket.created_at).toLocaleDateString(
                          "en-US",
                          { month: "short", day: "numeric", year: "numeric" }
                        )}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3">
                        <button
                          onClick={(e) => {
                            e.preventDefault();
                            setDeleteId(bucket.id);
                          }}
                          className="text-neutral-500 hover:text-red-400 transition-colors"
                          title="Delete bucket"
                        >
                          <svg
                            className="h-4 w-4"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                            strokeWidth={2}
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                            />
                          </svg>
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Create Modal */}
      {showCreate && (
        <Modal title="Create Bucket" onClose={() => setShowCreate(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreate();
            }}
            className="space-y-3"
          >
            {createError && (
              <div className="rounded-lg border border-amber-600/30 bg-amber-600/5 px-4 py-3">
                <p className="text-sm text-amber-400">{createError}</p>
                {createError.includes("Upgrade") && (
                  <Link
                    href="/billing"
                    className="mt-2 inline-block text-sm font-medium text-accent-400 hover:text-accent-300 transition-colors"
                  >
                    Go to Billing &rarr;
                  </Link>
                )}
              </div>
            )}
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Bucket Name
              </label>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="my-bucket"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">
                Access
              </label>
              <select
                value={formAccess}
                onChange={(e) => setFormAccess(e.target.value)}
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              >
                <option value="private">Private</option>
                <option value="public">Public</option>
              </select>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating}
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {creating ? "Creating..." : "Create"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Delete Confirmation Modal */}
      {deleteId && (
        <Modal title="Delete Bucket" onClose={() => setDeleteId(null)}>
          <p className="text-sm text-neutral-400 mb-4">
            Are you sure you want to delete this bucket? All objects inside will
            be permanently removed.
          </p>
          <div className="flex justify-end gap-2">
            <button
              onClick={() => setDeleteId(null)}
              className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors disabled:opacity-50"
            >
              {deleting ? "Deleting..." : "Delete"}
            </button>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
