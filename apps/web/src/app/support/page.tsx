"use client";

import { Shell } from "@/components/shell";
import { PageWithTableSkeleton } from "@/components/loading-skeleton";
import { ErrorState } from "@/components/error-state";
import { EmptyState } from "@/components/empty-state";
import { Modal } from "@/components/modal";
import { useApi } from "@/hooks/use-api";
import { type SupportTicket } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import Link from "next/link";
import { useState, useMemo } from "react";

const categoryOptions = [
  { value: "general", label: "General" },
  { value: "billing", label: "Billing" },
  { value: "technical", label: "Technical" },
  { value: "bug-report", label: "Bug Report" },
  { value: "feature-request", label: "Feature Request" },
];

const priorityOptions = [
  { value: "low", label: "Low" },
  { value: "normal", label: "Normal" },
  { value: "high", label: "High" },
  { value: "urgent", label: "Urgent" },
];

function statusColor(status: string): string {
  switch (status) {
    case "open":
      return "bg-blue-500/15 text-blue-400";
    case "in-progress":
      return "bg-yellow-500/15 text-yellow-400";
    case "waiting-on-customer":
      return "bg-orange-500/15 text-orange-400";
    case "resolved":
      return "bg-green-500/15 text-green-400";
    case "closed":
      return "bg-neutral-500/15 text-neutral-400";
    default:
      return "bg-neutral-500/15 text-neutral-400";
  }
}

function priorityColor(priority: string): string {
  switch (priority) {
    case "urgent":
      return "text-red-400";
    case "high":
      return "text-orange-400";
    case "normal":
      return "text-neutral-300";
    case "low":
      return "text-neutral-500";
    default:
      return "text-neutral-400";
  }
}

export default function SupportPage() {
  const { support, userPlan } = getApi();

  const {
    data: tickets,
    loading,
    error,
    refetch,
  } = useApi(() => support.list(), []);

  const { data: planData, loading: planLoading } = useApi(
    () => userPlan.get(),
    []
  );

  const [showCreate, setShowCreate] = useState(false);
  const [formSubject, setFormSubject] = useState("");
  const [formCategory, setFormCategory] = useState("general");
  const [formPriority, setFormPriority] = useState("normal");
  const [formMessage, setFormMessage] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");
  const [statusFilter, setStatusFilter] = useState("");

  const isFree = (planData?.tier ?? "free") === "free";
  const ticketList: SupportTicket[] = tickets ?? [];

  const filtered = useMemo(() => {
    if (!statusFilter) return ticketList;
    return ticketList.filter((t) => t.status === statusFilter);
  }, [ticketList, statusFilter]);

  const openCount = ticketList.filter((t) => t.status === "open").length;
  const inProgressCount = ticketList.filter(
    (t) => t.status === "in-progress" || t.status === "waiting-on-customer"
  ).length;
  const resolvedCount = ticketList.filter(
    (t) => t.status === "resolved" || t.status === "closed"
  ).length;

  const handleCreate = async () => {
    if (!formSubject.trim() || !formMessage.trim() || creating) return;
    setCreating(true);
    setCreateError("");
    try {
      await support.create({
        subject: formSubject.trim(),
        category: formCategory,
        priority: formPriority,
        message: formMessage.trim(),
      });
      setShowCreate(false);
      setFormSubject("");
      setFormCategory("general");
      setFormPriority("normal");
      setFormMessage("");
      refetch();
    } catch (err: unknown) {
      const status = (err as { status?: number }).status;
      if (status === 403) {
        setCreateError("Support requires a Pro plan or higher.");
      } else {
        setCreateError(
          err instanceof Error ? err.message : "Failed to create ticket"
        );
      }
    } finally {
      setCreating(false);
    }
  };

  if (loading || planLoading) {
    return (
      <Shell>
        <PageWithTableSkeleton cols={5} rows={3} />
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

  // Free tier — show upgrade prompt
  if (isFree) {
    return (
      <Shell>
        <div className="space-y-6">
          <div>
            <h1 className="text-lg font-semibold text-white">Support</h1>
            <p className="text-sm text-neutral-500">
              Get help from the Zenith team
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
                  d="M16.712 4.33a9.027 9.027 0 011.652 1.306c.51.51.944 1.064 1.306 1.652M16.712 4.33l-3.448 4.138m3.448-4.138a9.014 9.014 0 00-9.424 0M19.67 7.288l-4.138 3.448m4.138-3.448a9.014 9.014 0 010 9.424m-4.138-5.976a3.736 3.736 0 00-.88-1.388 3.737 3.737 0 00-1.388-.88m2.268 2.268a3.765 3.765 0 010 2.528m-2.268-4.796l-3.448 4.138m5.716-.37l-4.138 3.448m0 0a3.736 3.736 0 01-1.388.88 3.737 3.737 0 01-1.388-.88m2.776 0l-4.138 3.448m0 0a9.027 9.027 0 01-1.306-1.652m1.306 1.652a9.014 9.014 0 010-9.424m0 9.424l3.448-4.138m-3.448 4.138a9.027 9.027 0 01-1.652-1.306M4.33 16.712l4.138-3.448m-4.138 3.448a9.014 9.014 0 010-9.424m4.138 5.976a3.765 3.765 0 010-2.528m0 0a3.736 3.736 0 01.88-1.388 3.737 3.737 0 011.388-.88m-2.268 2.268L4.33 7.288m6.406 1.18L7.288 4.33m0 0a9.027 9.027 0 011.652-1.306"
                />
              </svg>
            </div>
            <h2 className="text-xl font-semibold text-white mb-2">
              Support is a Pro Feature
            </h2>
            <p className="text-sm text-neutral-400 text-center max-w-md mb-6">
              Get dedicated support from the Zenith team. Create tickets, track
              progress, and get timely responses.
            </p>
            <div className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-xs text-neutral-500 mb-8">
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Ticket support
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Priority routing
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Threaded conversations
              </span>
              <span className="flex items-center gap-1.5">
                <svg className="h-4 w-4 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                Email notifications
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
            <h1 className="text-lg font-semibold text-white">Support</h1>
            <p className="text-sm text-neutral-500">
              Get help from the Zenith team
            </p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="rounded-lg bg-accent-500 hover:bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            New Ticket
          </button>
        </div>

        {/* Stat cards */}
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: "Open", value: openCount, color: "text-blue-400" },
            { label: "In Progress", value: inProgressCount, color: "text-yellow-400" },
            { label: "Resolved", value: resolvedCount, color: "text-green-400" },
          ].map((stat) => (
            <div
              key={stat.label}
              className="rounded-xl border border-border bg-surface-100 p-4"
            >
              <p className="text-xs text-neutral-500">{stat.label}</p>
              <p className={`text-2xl font-semibold ${stat.color}`}>
                {stat.value}
              </p>
            </div>
          ))}
        </div>

        {/* Status filter */}
        <div className="flex gap-2">
          {["", "open", "in-progress", "waiting-on-customer", "resolved", "closed"].map(
            (s) => (
              <button
                key={s}
                onClick={() => setStatusFilter(s)}
                className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                  statusFilter === s
                    ? "bg-accent-500/15 text-accent-400"
                    : "text-neutral-400 hover:bg-surface-300 hover:text-white"
                }`}
              >
                {s === "" ? "All" : s.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase())}
              </button>
            )
          )}
        </div>

        {/* Ticket table */}
        {filtered.length === 0 ? (
          <EmptyState
            title="No tickets"
            description="You haven't created any support tickets yet."
          />
        ) : (
          <div className="overflow-hidden rounded-xl border border-border">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-surface-100 text-left text-xs font-medium text-neutral-500">
                  <th className="px-4 py-3">Subject</th>
                  <th className="px-4 py-3">Category</th>
                  <th className="px-4 py-3">Priority</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Updated</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {filtered.map((ticket) => (
                  <tr
                    key={ticket.id}
                    className="bg-surface-50 transition-colors hover:bg-surface-100"
                  >
                    <td className="px-4 py-3">
                      <Link
                        href={`/support/${ticket.id}`}
                        className="text-sm font-medium text-white hover:text-accent-400 transition-colors"
                      >
                        {ticket.subject}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-sm text-neutral-400 capitalize">
                      {ticket.category.replace(/-/g, " ")}
                    </td>
                    <td className={`px-4 py-3 text-sm capitalize ${priorityColor(ticket.priority)}`}>
                      {ticket.priority}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColor(ticket.status)}`}
                      >
                        {ticket.status.replace(/-/g, " ")}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-neutral-500">
                      {new Date(ticket.updated_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create ticket modal */}
      {showCreate && (
        <Modal title="New Support Ticket" onClose={() => setShowCreate(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-neutral-300 mb-1">
                Subject
              </label>
              <input
                value={formSubject}
                onChange={(e) => setFormSubject(e.target.value)}
                placeholder="Brief description of your issue"
                className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-1">
                  Category
                </label>
                <select
                  value={formCategory}
                  onChange={(e) => setFormCategory(e.target.value)}
                  className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white outline-none focus:border-accent-500"
                >
                  {categoryOptions.map((o) => (
                    <option key={o.value} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-1">
                  Priority
                </label>
                <select
                  value={formPriority}
                  onChange={(e) => setFormPriority(e.target.value)}
                  className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white outline-none focus:border-accent-500"
                >
                  {priorityOptions.map((o) => (
                    <option key={o.value} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-neutral-300 mb-1">
                Message
              </label>
              <textarea
                value={formMessage}
                onChange={(e) => setFormMessage(e.target.value)}
                placeholder="Describe your issue in detail..."
                rows={5}
                className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500 resize-none"
              />
            </div>
            {createError && (
              <p className="text-sm text-red-400">{createError}</p>
            )}
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={creating || !formSubject.trim() || !formMessage.trim()}
                className="rounded-lg bg-accent-500 hover:bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {creating ? "Creating..." : "Create Ticket"}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Shell>
  );
}
