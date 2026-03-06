"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { type StorageBucketV2, type StorageObject } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState, useCallback, useRef } from "react";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

function getFileName(key: string, prefix: string): string {
  const relative = key.startsWith(prefix) ? key.slice(prefix.length) : key;
  return relative.replace(/\/$/, "");
}

function getFileIcon(key: string, isFolder: boolean): string {
  if (isFolder) return "folder";
  const ext = key.split(".").pop()?.toLowerCase() ?? "";
  if (["jpg", "jpeg", "png", "gif", "svg", "webp", "ico"].includes(ext)) return "image";
  if (["mp4", "webm", "avi", "mov"].includes(ext)) return "video";
  if (["pdf"].includes(ext)) return "pdf";
  if (["zip", "tar", "gz", "rar", "7z"].includes(ext)) return "archive";
  if (["json", "xml", "yaml", "yml", "toml"].includes(ext)) return "config";
  if (["js", "ts", "py", "go", "rs", "java", "css", "html"].includes(ext)) return "code";
  return "file";
}

function FileIcon({ type }: { type: string }) {
  if (type === "folder") {
    return (
      <svg className="h-5 w-5 text-amber-400" fill="currentColor" viewBox="0 0 20 20">
        <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
      </svg>
    );
  }
  const colors: Record<string, string> = {
    image: "text-purple-400",
    video: "text-pink-400",
    pdf: "text-red-400",
    archive: "text-orange-400",
    config: "text-cyan-400",
    code: "text-green-400",
    file: "text-neutral-400",
  };
  return (
    <svg className={`h-5 w-5 ${colors[type] ?? "text-neutral-400"}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
    </svg>
  );
}

export default function BucketDetailPage() {
  const params = useParams();
  const router = useRouter();
  const bucketId = params.id as string;
  const { storageBuckets } = getApi();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [tab, setTab] = useState<"objects" | "settings">("objects");
  const [prefix, setPrefix] = useState("");
  const [uploading, setUploading] = useState(false);
  const [showCreateFolder, setShowCreateFolder] = useState(false);
  const [folderName, setFolderName] = useState("");
  const [creatingFolder, setCreatingFolder] = useState(false);
  const [showDeleteBucket, setShowDeleteBucket] = useState(false);
  const [deletingBucket, setDeletingBucket] = useState(false);
  const [savingAccess, setSavingAccess] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [deletingObjects, setDeletingObjects] = useState(false);

  const {
    data: bucket,
    loading: bucketLoading,
    error: bucketError,
    refetch: refetchBucket,
  } = useApi(() => storageBuckets.get(bucketId), [bucketId]);

  const {
    data: objectsData,
    loading: objectsLoading,
    refetch: refetchObjects,
  } = useApi(() => storageBuckets.listObjects(bucketId, prefix), [bucketId, prefix]);

  const objects: StorageObject[] = objectsData?.objects ?? [];

  const navigateToFolder = useCallback((folderPrefix: string) => {
    setPrefix(folderPrefix);
    setSelectedKeys(new Set());
  }, []);

  // Breadcrumb segments
  const breadcrumbs = prefix
    .split("/")
    .filter(Boolean)
    .map((segment, i, arr) => ({
      label: segment,
      prefix: arr.slice(0, i + 1).join("/") + "/",
    }));

  const handleUpload = async (files: FileList | null) => {
    if (!files || files.length === 0 || !bucket) return;
    setUploading(true);
    try {
      for (const file of Array.from(files)) {
        const key = prefix + file.name;
        await storageBuckets.uploadObject(bucketId, key, file);
      }
      refetchObjects();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const handleDownload = async (key: string) => {
    try {
      await storageBuckets.downloadObject(bucketId, key);
    } catch (err) {
      alert(err instanceof Error ? err.message : "Download failed");
    }
  };

  const handleDeleteObject = async (key: string) => {
    try {
      await storageBuckets.deleteObject(bucketId, key);
      refetchObjects();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Delete failed");
    }
  };

  const handleDeleteSelected = async () => {
    if (selectedKeys.size === 0) return;
    setDeletingObjects(true);
    try {
      for (const key of selectedKeys) {
        await storageBuckets.deleteObject(bucketId, key);
      }
      setSelectedKeys(new Set());
      refetchObjects();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Delete failed");
    } finally {
      setDeletingObjects(false);
    }
  };

  const handleCreateFolder = async () => {
    if (!folderName.trim() || creatingFolder) return;
    setCreatingFolder(true);
    try {
      await storageBuckets.createFolder(bucketId, prefix + folderName.trim());
      setShowCreateFolder(false);
      setFolderName("");
      refetchObjects();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to create folder");
    } finally {
      setCreatingFolder(false);
    }
  };

  const handleDeleteBucket = async () => {
    if (deletingBucket) return;
    setDeletingBucket(true);
    try {
      await storageBuckets.delete(bucketId);
      router.push("/storage");
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete bucket");
      setDeletingBucket(false);
    }
  };

  const handleSaveAccess = async (newAccess: string) => {
    setSavingAccess(true);
    try {
      await storageBuckets.update(bucketId, { access: newAccess });
      refetchBucket();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to update access");
    } finally {
      setSavingAccess(false);
    }
  };

  const toggleSelect = (key: string) => {
    setSelectedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  if (bucketLoading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={5} rows={4} />
      </Shell>
    );
  }

  if (bucketError || !bucket) {
    return (
      <Shell>
        <ErrorState message={bucketError || "Bucket not found"} onRetry={refetchBucket} />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link href="/storage" className="text-neutral-500 hover:text-white transition-colors">
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
              </svg>
            </Link>
            <div>
              <h1 className="text-lg font-semibold text-white">{bucket.name}</h1>
              <p className="text-sm text-neutral-500">
                {bucket.objects} objects &middot; {formatBytes(bucket.size_mb * 1024 * 1024)} &middot; {bucket.region}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                bucket.access === "private"
                  ? "bg-neutral-500/10 text-neutral-400"
                  : "bg-amber-500/10 text-amber-400"
              }`}
            >
              {bucket.access === "private" ? "Private" : "Public"}
            </span>
            <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-400">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
              {bucket.status}
            </span>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-border">
          <button
            onClick={() => setTab("objects")}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 ${
              tab === "objects"
                ? "text-white border-accent-500"
                : "text-neutral-500 border-transparent hover:text-neutral-300"
            }`}
          >
            Objects
          </button>
          <button
            onClick={() => setTab("settings")}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 ${
              tab === "settings"
                ? "text-white border-accent-500"
                : "text-neutral-500 border-transparent hover:text-neutral-300"
            }`}
          >
            Settings
          </button>
        </div>

        {/* Objects Tab */}
        {tab === "objects" && (
          <div className="space-y-4">
            {/* Breadcrumb */}
            <div className="flex items-center gap-1 text-sm">
              <button
                onClick={() => navigateToFolder("")}
                className={`hover:text-accent-400 transition-colors ${
                  prefix === "" ? "text-white font-medium" : "text-neutral-500"
                }`}
              >
                {bucket.name}
              </button>
              {breadcrumbs.map((crumb) => (
                <span key={crumb.prefix} className="flex items-center gap-1">
                  <span className="text-neutral-600">/</span>
                  <button
                    onClick={() => navigateToFolder(crumb.prefix)}
                    className={`hover:text-accent-400 transition-colors ${
                      prefix === crumb.prefix ? "text-white font-medium" : "text-neutral-500"
                    }`}
                  >
                    {crumb.label}
                  </button>
                </span>
              ))}
            </div>

            {/* Toolbar */}
            <div className="flex items-center gap-2">
              <input
                ref={fileInputRef}
                type="file"
                multiple
                className="hidden"
                onChange={(e) => handleUpload(e.target.files)}
              />
              <button
                onClick={() => fileInputRef.current?.click()}
                disabled={uploading}
                className="rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors disabled:opacity-50"
              >
                {uploading ? "Uploading..." : "Upload"}
              </button>
              <button
                onClick={() => setShowCreateFolder(true)}
                className="rounded-lg border border-border px-3 py-1.5 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                New Folder
              </button>
              {selectedKeys.size > 0 && (
                <button
                  onClick={handleDeleteSelected}
                  disabled={deletingObjects}
                  className="rounded-lg bg-red-600/10 border border-red-600/30 px-3 py-1.5 text-sm text-red-400 hover:bg-red-600/20 transition-colors disabled:opacity-50"
                >
                  {deletingObjects ? "Deleting..." : `Delete (${selectedKeys.size})`}
                </button>
              )}
            </div>

            {/* Object table */}
            {objectsLoading ? (
              <PageWithTableSkeleton cols={5} rows={4} />
            ) : objects.length === 0 ? (
              <div className="flex flex-col items-center justify-center rounded-lg border border-border bg-surface-100 py-12">
                <svg className="h-12 w-12 text-neutral-600 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                </svg>
                <p className="text-sm text-neutral-500">This folder is empty</p>
                <p className="text-xs text-neutral-600 mt-1">Upload files or create folders to get started</p>
              </div>
            ) : (
              <div className="overflow-hidden rounded-lg border border-border">
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead>
                      <tr className="border-b border-border bg-surface-100">
                        <th className="w-10 px-4 py-3">
                          <input
                            type="checkbox"
                            checked={selectedKeys.size === objects.filter((o) => !o.is_folder).length && objects.filter((o) => !o.is_folder).length > 0}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedKeys(new Set(objects.filter((o) => !o.is_folder).map((o) => o.key)));
                              } else {
                                setSelectedKeys(new Set());
                              }
                            }}
                            className="rounded border-border bg-surface-200"
                          />
                        </th>
                        <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Name</th>
                        <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Size</th>
                        <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500">Last Modified</th>
                        <th className="whitespace-nowrap px-4 py-3 text-xs font-medium text-neutral-500"></th>
                      </tr>
                    </thead>
                    <tbody>
                      {objects.map((obj) => {
                        const name = getFileName(obj.key, prefix);
                        const iconType = getFileIcon(obj.key, obj.is_folder);
                        return (
                          <tr
                            key={obj.key}
                            className="border-b border-border last:border-0 hover:bg-surface-200 transition-colors"
                          >
                            <td className="w-10 px-4 py-3">
                              {!obj.is_folder && (
                                <input
                                  type="checkbox"
                                  checked={selectedKeys.has(obj.key)}
                                  onChange={() => toggleSelect(obj.key)}
                                  className="rounded border-border bg-surface-200"
                                />
                              )}
                            </td>
                            <td className="whitespace-nowrap px-4 py-3">
                              <div className="flex items-center gap-2">
                                <FileIcon type={iconType} />
                                {obj.is_folder ? (
                                  <button
                                    onClick={() => navigateToFolder(obj.key)}
                                    className="font-medium text-white hover:text-accent-400 transition-colors"
                                  >
                                    {name}
                                  </button>
                                ) : (
                                  <span className="text-neutral-300">{name}</span>
                                )}
                              </div>
                            </td>
                            <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                              {obj.is_folder ? "-" : formatBytes(obj.size)}
                            </td>
                            <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-neutral-400">
                              {obj.is_folder || !obj.last_modified
                                ? "-"
                                : new Date(obj.last_modified).toLocaleString("en-US", {
                                    month: "short",
                                    day: "numeric",
                                    year: "numeric",
                                    hour: "2-digit",
                                    minute: "2-digit",
                                  })}
                            </td>
                            <td className="whitespace-nowrap px-4 py-3">
                              {!obj.is_folder && (
                                <div className="flex items-center gap-2">
                                  <button
                                    onClick={() => handleDownload(obj.key)}
                                    className="text-neutral-500 hover:text-accent-400 transition-colors"
                                    title="Download"
                                  >
                                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                      <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                                    </svg>
                                  </button>
                                  <button
                                    onClick={() => handleDeleteObject(obj.key)}
                                    className="text-neutral-500 hover:text-red-400 transition-colors"
                                    title="Delete"
                                  >
                                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                      <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                    </svg>
                                  </button>
                                </div>
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Settings Tab */}
        {tab === "settings" && (
          <div className="space-y-6">
            {/* Access Control */}
            <div className="rounded-lg border border-border bg-surface-100 p-6">
              <h3 className="text-sm font-medium text-white mb-4">Access Control</h3>
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="access"
                    value="private"
                    checked={bucket.access === "private"}
                    onChange={() => handleSaveAccess("private")}
                    disabled={savingAccess}
                    className="text-accent-500"
                  />
                  <span className="text-sm text-neutral-300">Private</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="access"
                    value="public"
                    checked={bucket.access === "public"}
                    onChange={() => handleSaveAccess("public")}
                    disabled={savingAccess}
                    className="text-accent-500"
                  />
                  <span className="text-sm text-neutral-300">Public</span>
                </label>
                {savingAccess && (
                  <span className="text-xs text-neutral-500">Saving...</span>
                )}
              </div>
            </div>

            {/* Bucket Info */}
            <div className="rounded-lg border border-border bg-surface-100 p-6">
              <h3 className="text-sm font-medium text-white mb-4">Bucket Info</h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-neutral-500">Endpoint</span>
                  <div className="flex items-center gap-2">
                    <code className="text-xs text-neutral-300 bg-surface-200 px-2 py-1 rounded">{bucket.endpoint}</code>
                    <button
                      onClick={() => navigator.clipboard.writeText(bucket.endpoint)}
                      className="text-neutral-500 hover:text-white transition-colors"
                      title="Copy"
                    >
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                    </button>
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-neutral-500">Region</span>
                  <span className="text-sm text-neutral-300 font-mono">{bucket.region}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-neutral-500">Created</span>
                  <span className="text-sm text-neutral-300">
                    {new Date(bucket.created_at).toLocaleString("en-US", {
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-neutral-500">Size</span>
                  <span className="text-sm text-neutral-300">{formatBytes(bucket.size_mb * 1024 * 1024)} / {formatBytes(bucket.max_size_mb * 1024 * 1024)}</span>
                </div>
              </div>
            </div>

            {/* Danger Zone */}
            <div className="rounded-lg border border-red-600/30 bg-red-600/5 p-6">
              <h3 className="text-sm font-medium text-red-400 mb-2">Danger Zone</h3>
              <p className="text-sm text-neutral-500 mb-4">
                Deleting this bucket will permanently remove all objects inside.
              </p>
              <button
                onClick={() => setShowDeleteBucket(true)}
                className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors"
              >
                Delete Bucket
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Create Folder Modal */}
      {showCreateFolder && (
        <Modal title="Create Folder" onClose={() => setShowCreateFolder(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreateFolder();
            }}
            className="space-y-3"
          >
            <div>
              <label className="mb-1 block text-xs font-medium text-neutral-400">Folder Name</label>
              <input
                type="text"
                value={folderName}
                onChange={(e) => setFolderName(e.target.value)}
                placeholder="my-folder"
                className="w-full rounded-md border border-border bg-surface-200 px-3 py-2 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
                required
              />
              {prefix && (
                <p className="mt-1 text-xs text-neutral-600">
                  Will be created at: {prefix}{folderName}/
                </p>
              )}
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <button
                type="button"
                onClick={() => setShowCreateFolder(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creatingFolder}
                className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {creatingFolder ? "Creating..." : "Create"}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Delete Bucket Confirmation */}
      {showDeleteBucket && (
        <Modal title="Delete Bucket" onClose={() => setShowDeleteBucket(false)}>
          <p className="text-sm text-neutral-400 mb-4">
            Are you sure you want to delete <strong className="text-white">{bucket.name}</strong>?
            All objects inside will be permanently removed. This action cannot be undone.
          </p>
          <div className="flex justify-end gap-2">
            <button
              onClick={() => setShowDeleteBucket(false)}
              className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleDeleteBucket}
              disabled={deletingBucket}
              className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors disabled:opacity-50"
            >
              {deletingBucket ? "Deleting..." : "Delete Bucket"}
            </button>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
