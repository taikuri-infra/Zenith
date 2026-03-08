"use client";
import { useState } from "react";
import { Shell } from "@/components/shell";
import { StatusBadge } from "@/components/status-badge";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Modal } from "@/components/modal";
import { Plus, Trash2, Loader2, Eye, EyeOff, Copy, Check, AlertTriangle, ExternalLink, Lock } from "lucide-react";
import { useApi } from "@/hooks/use-api";
import { useToast } from "@/components/toast";
import { useProject } from "@/hooks/use-project";
import { getApi, isDemoMode } from "@/lib/get-api";
import { AppDatabase } from "@/lib/api";
import Link from "next/link";

function CopyField({ label, value, masked = false, mono = true }: { label: string; value: string; masked?: boolean; mono?: boolean }) {
  const [copied, setCopied] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const displayValue = masked && !revealed ? "••••••••••••••••" : value;
  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div>
      <label className="mb-1 block text-[11px] font-medium text-neutral-500">{label}</label>
      <div className="flex items-center gap-2">
        <div className="flex-1 rounded-md bg-surface-200 px-3 py-2">
          <code className={`text-xs text-white break-all ${mono ? "font-mono" : ""}`}>{displayValue}</code>
        </div>
        {masked && (
          <button
            onClick={() => setRevealed(!revealed)}
            className="shrink-0 rounded-md border border-border p-2 text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
          >
            {revealed ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
          </button>
        )}
        <button
          onClick={handleCopy}
          className="shrink-0 rounded-md border border-border p-2 text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
        >
          {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
        </button>
      </div>
    </div>
  );
}

export default function QueuesPage() {
  const { toast } = useToast();
  const { standaloneDatabases, userPlan } = getApi();
  const projectId = useProject();

  // RabbitMQ state
  const [showCreate, setShowCreate] = useState(false);
  const [createName, setCreateName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createdQueue, setCreatedQueue] = useState<AppDatabase | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  // Kafka state
  const [showCreateKafka, setShowCreateKafka] = useState(false);
  const [createKafkaName, setCreateKafkaName] = useState("");
  const [creatingKafka, setCreatingKafka] = useState(false);
  const [createdKafka, setCreatedKafka] = useState<AppDatabase | null>(null);
  const [confirmDeleteKafkaId, setConfirmDeleteKafkaId] = useState<string | null>(null);

  const {
    data: dbList,
    loading,
    error,
    refetch,
  } = useApi(() => standaloneDatabases.list(projectId || undefined), [projectId]);

  const { data: planData } = useApi(() => userPlan.get(), []);

  const tier = planData?.tier ?? "free";
  const kafkaAllowed = tier === "business" || tier === "enterprise";

  const queues: AppDatabase[] = (dbList || []).filter((d) => d.engine === "rabbitmq");
  const kafkaClusters: AppDatabase[] = (dbList || []).filter((d) => d.engine === "kafka");
  const readyCount = queues.filter((q) => q.status === "ready").length;
  const kafkaReadyCount = kafkaClusters.filter((k) => k.status === "ready").length;
  const totalCount = queues.length + kafkaClusters.length;
  const totalReady = readyCount + kafkaReadyCount;

  const handleCreate = async () => {
    if (!createName.trim()) return;
    setCreating(true);
    try {
      const result = await standaloneDatabases.create({ name: createName.trim(), engine: "rabbitmq" });
      setShowCreate(false);
      setCreateName("");
      setCreatedQueue(result);
      refetch();
    } catch {
      toast("error", "Failed to create queue");
    } finally {
      setCreating(false);
    }
  };

  const handleCreateKafka = async () => {
    if (!createKafkaName.trim()) return;
    setCreatingKafka(true);
    try {
      const result = await standaloneDatabases.create({ name: createKafkaName.trim(), engine: "kafka" });
      setShowCreateKafka(false);
      setCreateKafkaName("");
      setCreatedKafka(result);
      refetch();
    } catch {
      toast("error", "Failed to create Kafka cluster");
    } finally {
      setCreatingKafka(false);
    }
  };

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await standaloneDatabases.delete(id);
      refetch();
    } catch {
      toast("error", "Failed to delete queue");
    } finally {
      setDeletingId(null);
      setConfirmDeleteId(null);
      setConfirmDeleteKafkaId(null);
    }
  };

  if (loading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={4} rows={3} />
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

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Queues</h1>
            <p className="text-sm text-neutral-500">
              {totalCount > 0
                ? `${totalCount} instance${totalCount !== 1 ? "s" : ""}, ${totalReady} ready`
                : "Managed message queues for async workloads"}
            </p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 rounded-lg bg-accent-500 hover:bg-accent-600 text-white px-3 py-1.5 text-sm transition-colors"
          >
            <Plus className="h-4 w-4" />
            Create Queue
          </button>
        </div>

        {/* RabbitMQ Section */}
        <div>
          <p className="mb-3 text-xs font-medium text-neutral-600 uppercase tracking-wider">RabbitMQ</p>
          {queues.length === 0 ? (
            <EmptyState
              title="No queues yet"
              description="Create a RabbitMQ queue to start sending and receiving messages."
              actionLabel="Create Queue"
              onAction={() => setShowCreate(true)}
            />
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {queues.map((queue) => (
                <div
                  key={queue.id}
                  className="group relative rounded-xl border border-border bg-surface-100 p-5 hover:border-neutral-600 transition-colors"
                >
                  <div className="flex items-start justify-between gap-3">
                    <Link href={`/databases/${queue.id}`} className="flex items-center gap-3 min-w-0 flex-1">
                      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-purple-500/20 text-sm font-bold text-purple-400">
                        Q
                      </div>
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-white group-hover:text-accent-400 transition-colors">
                          {queue.name}
                        </p>
                        <p className="truncate text-xs text-neutral-500 font-mono">{queue.host}</p>
                      </div>
                    </Link>
                    <button
                      onClick={() => setConfirmDeleteId(queue.id)}
                      disabled={deletingId === queue.id}
                      className="shrink-0 rounded-md p-1.5 text-neutral-600 hover:text-red-400 hover:bg-red-500/10 transition-colors disabled:opacity-50"
                      title="Delete queue"
                    >
                      {deletingId === queue.id ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Trash2 className="h-4 w-4" />
                      )}
                    </button>
                  </div>

                  <div className="mt-4 flex items-center justify-between">
                    <StatusBadge status={queue.status as "ready" | "provisioning" | "error" | "deleting"} />
                    <Link
                      href={`/databases/${queue.id}`}
                      className="flex items-center gap-1 text-xs text-neutral-500 hover:text-accent-400 transition-colors"
                    >
                      <ExternalLink className="h-3 w-3" />
                      Details
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Kafka Section */}
        <div>
          <div className="mb-3 flex items-center gap-2">
            <p className="text-xs font-medium text-neutral-600 uppercase tracking-wider">Kafka</p>
            {!kafkaAllowed && (
              <span className="inline-flex items-center gap-1 rounded bg-amber-500/15 px-1.5 py-0.5 text-[9px] font-medium text-amber-400">
                <Lock className="h-2.5 w-2.5" />
                Business+ plan required
              </span>
            )}
          </div>

          {!kafkaAllowed ? (
            <div className="rounded-xl border border-border bg-surface-100 p-5">
              <div className="flex items-center gap-3 mb-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-orange-500/20 text-sm font-bold text-orange-400">
                  K
                </div>
                <div>
                  <h3 className="text-sm font-medium text-white">Apache Kafka</h3>
                  <p className="text-xs text-neutral-500">Distributed event streaming platform</p>
                </div>
              </div>
              <p className="text-xs text-neutral-500">
                High-throughput, low-latency event streaming with topic partitioning, consumer groups, and exactly-once semantics.
                Upgrade to the Business plan or higher to provision Kafka clusters.
              </p>
              <Link
                href="/billing"
                className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg border border-amber-500/30 bg-amber-500/5 py-2 text-xs font-medium text-amber-400 hover:bg-amber-500/10 transition-colors"
              >
                <Lock className="h-3 w-3" />
                Upgrade to Business
              </Link>
            </div>
          ) : kafkaClusters.length === 0 ? (
            <EmptyState
              title="No Kafka clusters yet"
              description="Create a Kafka cluster for high-throughput event streaming."
              actionLabel="Create Kafka Cluster"
              onAction={() => setShowCreateKafka(true)}
            />
          ) : (
            <>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {kafkaClusters.map((cluster) => (
                  <div
                    key={cluster.id}
                    className="group relative rounded-xl border border-border bg-surface-100 p-5 hover:border-neutral-600 transition-colors"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <Link href={`/databases/${cluster.id}`} className="flex items-center gap-3 min-w-0 flex-1">
                        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-orange-500/20 text-sm font-bold text-orange-400">
                          K
                        </div>
                        <div className="min-w-0">
                          <p className="truncate text-sm font-medium text-white group-hover:text-accent-400 transition-colors">
                            {cluster.name}
                          </p>
                          <p className="truncate text-xs text-neutral-500 font-mono">{cluster.host}:{cluster.port}</p>
                        </div>
                      </Link>
                      <button
                        onClick={() => setConfirmDeleteKafkaId(cluster.id)}
                        disabled={deletingId === cluster.id}
                        className="shrink-0 rounded-md p-1.5 text-neutral-600 hover:text-red-400 hover:bg-red-500/10 transition-colors disabled:opacity-50"
                        title="Delete cluster"
                      >
                        {deletingId === cluster.id ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Trash2 className="h-4 w-4" />
                        )}
                      </button>
                    </div>

                    <div className="mt-4 flex items-center justify-between">
                      <StatusBadge status={cluster.status as "ready" | "provisioning" | "error" | "deleting"} />
                      <Link
                        href={`/databases/${cluster.id}`}
                        className="flex items-center gap-1 text-xs text-neutral-500 hover:text-accent-400 transition-colors"
                      >
                        <ExternalLink className="h-3 w-3" />
                        Details
                      </Link>
                    </div>
                  </div>
                ))}
              </div>
              <div className="mt-3 flex justify-end">
                <button
                  onClick={() => setShowCreateKafka(true)}
                  className="flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-xs text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
                >
                  <Plus className="h-3.5 w-3.5" />
                  Create Kafka Cluster
                </button>
              </div>
            </>
          )}
        </div>

        {/* NATS — Coming Soon */}
        <div>
          <p className="mb-3 text-xs font-medium text-neutral-600 uppercase tracking-wider">Coming Soon</p>
          <div className="rounded-xl border border-border bg-surface-100 p-5 opacity-50">
            <div className="flex items-center gap-3 mb-3">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface-300 text-sm font-bold text-neutral-400">
                N
              </div>
              <div>
                <h3 className="text-sm font-medium text-white">NATS</h3>
                <span className="rounded bg-neutral-500/20 px-1.5 py-0.5 text-[9px] text-neutral-500">
                  Coming Soon
                </span>
              </div>
            </div>
            <p className="text-xs text-neutral-500">
              Cloud-native messaging with JetStream persistence and at-least-once delivery.
            </p>
            <button
              disabled
              className="mt-4 w-full rounded-lg border border-border bg-surface-200 py-2 text-xs text-neutral-500 cursor-not-allowed"
            >
              Create Queue
            </button>
          </div>
        </div>
      </div>

      {/* Create RabbitMQ Queue Modal */}
      {showCreate && (
        <Modal title="Create RabbitMQ Queue" onClose={() => { setShowCreate(false); setCreateName(""); }}>
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg border border-purple-500/30 bg-purple-500/5 px-4 py-3">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-purple-500/20 text-sm font-bold text-purple-400">
                Q
              </div>
              <div>
                <p className="text-sm font-medium text-white">RabbitMQ</p>
                <p className="text-xs text-neutral-500">AMQP message broker with routing, queuing, and pub/sub</p>
              </div>
            </div>

            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-400">Queue Name</label>
              <input
                type="text"
                value={createName}
                onChange={(e) => setCreateName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                placeholder="my-queue"
                autoFocus
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
              <p className="mt-1 text-[11px] text-neutral-600">Lowercase letters, numbers, and hyphens only</p>
            </div>

            <div className="flex justify-end gap-2">
              <button
                onClick={() => { setShowCreate(false); setCreateName(""); }}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={!createName.trim() || creating}
                className="flex items-center gap-2 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {creating && <Loader2 className="h-4 w-4 animate-spin" />}
                {creating ? "Creating..." : "Create"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Create Kafka Cluster Modal */}
      {showCreateKafka && (
        <Modal title="Create Kafka Cluster" onClose={() => { setShowCreateKafka(false); setCreateKafkaName(""); }}>
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg border border-orange-500/30 bg-orange-500/5 px-4 py-3">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-orange-500/20 text-sm font-bold text-orange-400">
                K
              </div>
              <div>
                <p className="text-sm font-medium text-white">Apache Kafka</p>
                <p className="text-xs text-neutral-500">Distributed event streaming with topic partitioning and consumer groups</p>
              </div>
            </div>

            <div>
              <label className="mb-1.5 block text-xs font-medium text-neutral-400">Cluster Name</label>
              <input
                type="text"
                value={createKafkaName}
                onChange={(e) => setCreateKafkaName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                onKeyDown={(e) => e.key === "Enter" && handleCreateKafka()}
                placeholder="my-kafka-cluster"
                autoFocus
                className="w-full rounded-lg border border-border bg-surface-200 px-3 py-2.5 text-sm text-white placeholder:text-neutral-600 focus:border-accent-500 focus:outline-none"
              />
              <p className="mt-1 text-[11px] text-neutral-600">Lowercase letters, numbers, and hyphens only</p>
            </div>

            <div className="flex justify-end gap-2">
              <button
                onClick={() => { setShowCreateKafka(false); setCreateKafkaName(""); }}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateKafka}
                disabled={!createKafkaName.trim() || creatingKafka}
                className="flex items-center gap-2 rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
              >
                {creatingKafka && <Loader2 className="h-4 w-4 animate-spin" />}
                {creatingKafka ? "Creating..." : "Create"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Delete Confirmation Modal (RabbitMQ) */}
      {confirmDeleteId && (() => {
        const queue = queues.find((q) => q.id === confirmDeleteId);
        if (!queue) return null;
        return (
          <Modal title="Delete Queue" onClose={() => setConfirmDeleteId(null)}>
            <div className="space-y-4">
              <div className="flex items-start gap-3 rounded-lg border border-red-500/30 bg-red-500/5 px-4 py-3">
                <AlertTriangle className="h-4 w-4 text-red-400 mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-red-300">This action cannot be undone</p>
                  <p className="text-xs text-red-400/70 mt-0.5">
                    Deleting <span className="font-mono font-semibold">{queue.name}</span> will permanently remove
                    the queue and all its data.
                  </p>
                </div>
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setConfirmDeleteId(null)}
                  className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleDelete(confirmDeleteId)}
                  disabled={deletingId === confirmDeleteId}
                  className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors disabled:opacity-50"
                >
                  {deletingId === confirmDeleteId && <Loader2 className="h-4 w-4 animate-spin" />}
                  {deletingId === confirmDeleteId ? "Deleting..." : "Delete Queue"}
                </button>
              </div>
            </div>
          </Modal>
        );
      })()}

      {/* Delete Confirmation Modal (Kafka) */}
      {confirmDeleteKafkaId && (() => {
        const cluster = kafkaClusters.find((k) => k.id === confirmDeleteKafkaId);
        if (!cluster) return null;
        return (
          <Modal title="Delete Kafka Cluster" onClose={() => setConfirmDeleteKafkaId(null)}>
            <div className="space-y-4">
              <div className="flex items-start gap-3 rounded-lg border border-red-500/30 bg-red-500/5 px-4 py-3">
                <AlertTriangle className="h-4 w-4 text-red-400 mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-red-300">This action cannot be undone</p>
                  <p className="text-xs text-red-400/70 mt-0.5">
                    Deleting <span className="font-mono font-semibold">{cluster.name}</span> will permanently remove
                    the Kafka cluster, all topics, and stored messages.
                  </p>
                </div>
              </div>
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setConfirmDeleteKafkaId(null)}
                  className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white hover:border-neutral-500 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleDelete(confirmDeleteKafkaId)}
                  disabled={deletingId === confirmDeleteKafkaId}
                  className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors disabled:opacity-50"
                >
                  {deletingId === confirmDeleteKafkaId && <Loader2 className="h-4 w-4 animate-spin" />}
                  {deletingId === confirmDeleteKafkaId ? "Deleting..." : "Delete Cluster"}
                </button>
              </div>
            </div>
          </Modal>
        );
      })()}

      {/* RabbitMQ Credentials Modal — shown after successful creation */}
      {createdQueue && (() => {
        const password = createdQueue.db_password ?? "";
        const vhost = createdQueue.db_name || "/";
        const connStr =
          createdQueue.connection_string ||
          `amqp://${createdQueue.db_user}:${password}@${createdQueue.host}:${createdQueue.port}/${encodeURIComponent(vhost)}`;

        return (
          <Modal title="Queue Created" onClose={() => setCreatedQueue(null)}>
            <div className="space-y-4">
              {/* Warning banner */}
              <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3">
                <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-amber-300">Save your credentials now</p>
                  <p className="text-xs text-amber-400/70 mt-0.5">
                    The password will not be shown again. Make sure to copy before closing.
                  </p>
                </div>
              </div>

              <CopyField label="Host" value={createdQueue.host} />
              <CopyField label="Port" value={String(createdQueue.port)} />
              <CopyField label="Virtual Host (vhost)" value={vhost} />
              <CopyField label="Username" value={createdQueue.db_user} />
              <CopyField label="Password" value={password} masked />
              <CopyField label="Connection String" value={connStr} />

              <div className="flex justify-end pt-2">
                <button
                  onClick={() => setCreatedQueue(null)}
                  className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
                >
                  I&apos;ve saved my credentials
                </button>
              </div>
            </div>
          </Modal>
        );
      })()}

      {/* Kafka Credentials Modal — shown after successful creation */}
      {createdKafka && (() => {
        const password = createdKafka.db_password ?? "";
        const bootstrap = `${createdKafka.host}:${createdKafka.port}`;
        const connStr =
          createdKafka.connection_string ||
          `kafka://${createdKafka.db_user}:${password}@${bootstrap}`;

        return (
          <Modal title="Kafka Cluster Created" onClose={() => setCreatedKafka(null)}>
            <div className="space-y-4">
              {/* Warning banner */}
              <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3">
                <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-amber-300">Save your credentials now</p>
                  <p className="text-xs text-amber-400/70 mt-0.5">
                    The password will not be shown again. Make sure to copy before closing.
                  </p>
                </div>
              </div>

              <CopyField label="Bootstrap Broker" value={bootstrap} />
              <CopyField label="Username" value={createdKafka.db_user} />
              <CopyField label="Password" value={password} masked />
              <CopyField label="Connection String" value={connStr} />

              <div className="flex justify-end pt-2">
                <button
                  onClick={() => setCreatedKafka(null)}
                  className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white hover:bg-accent-600 transition-colors"
                >
                  I&apos;ve saved my credentials
                </button>
              </div>
            </div>
          </Modal>
        );
      })()}
    </Shell>
  );
}
