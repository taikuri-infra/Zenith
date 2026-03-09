"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Shell } from "@/components/shell";
import { getApi } from "@/lib/get-api";
import type { SupportTicket } from "@/lib/api";

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
      return "bg-red-500/15 text-red-400";
    case "high":
      return "bg-orange-500/15 text-orange-400";
    case "normal":
      return "bg-neutral-500/15 text-neutral-300";
    case "low":
      return "bg-neutral-500/10 text-neutral-500";
    default:
      return "bg-neutral-500/15 text-neutral-400";
  }
}

const statusTabs = [
  { value: "", label: "All" },
  { value: "open", label: "Open" },
  { value: "in-progress", label: "In Progress" },
  { value: "waiting-on-customer", label: "Waiting" },
  { value: "resolved", label: "Resolved" },
  { value: "closed", label: "Closed" },
];

function ticketNumber(id: string): string {
  const num = parseInt(id.replace(/\D/g, "").slice(-6) || "0", 10) % 100000;
  return `#${String(num || 1).padStart(4, "0")}`;
}

export default function AdminSupportPage() {
  const apiClient = getApi();
  const [tickets, setTickets] = useState<SupportTicket[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const fetchTickets = async () => {
    setLoading(true);
    setError("");
    try {
      const res = await apiClient.support.list({
        status: statusFilter || undefined,
        limit,
        offset,
      });
      setTickets(res.items ?? []);
      setTotal(res.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load tickets");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTickets();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter, offset]);

  const totalPages = Math.ceil(total / limit);
  const currentPage = Math.floor(offset / limit) + 1;

  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Support Tickets</h1>
          <p className="text-sm text-neutral-500">
            {total > 0 ? `${total} tickets` : "Manage customer support requests"}
          </p>
        </div>

        {/* Status filter tabs */}
        <div className="flex gap-2">
          {statusTabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => {
                setStatusFilter(tab.value);
                setOffset(0);
              }}
              className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                statusFilter === tab.value
                  ? "bg-accent-600/15 text-accent-400"
                  : "text-neutral-400 hover:bg-surface-300 hover:text-white"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {error && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-400">
            {error}
          </div>
        )}

        {loading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="h-12 animate-pulse rounded-lg bg-surface-200"
              />
            ))}
          </div>
        ) : tickets.length === 0 ? (
          <div className="rounded-xl border border-border bg-surface-100 py-12 text-center text-sm text-neutral-500">
            No tickets found
          </div>
        ) : (
          <>
            <div className="overflow-hidden rounded-xl border border-border">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border bg-surface-100 text-left text-xs font-medium text-neutral-500">
                    <th className="px-4 py-3 w-20">Ticket</th>
                    <th className="px-4 py-3">Subject</th>
                    <th className="px-4 py-3">User</th>
                    <th className="px-4 py-3">Category</th>
                    <th className="px-4 py-3">Priority</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {tickets.map((ticket) => (
                    <tr
                      key={ticket.id}
                      className="bg-surface-50 transition-colors hover:bg-surface-100"
                    >
                      <td className="px-4 py-3 font-mono text-xs text-neutral-500">
                        {ticketNumber(ticket.id)}
                      </td>
                      <td className="px-4 py-3">
                        <Link
                          href={`/support/${ticket.id}`}
                          className="text-sm font-medium text-white hover:text-accent-400 transition-colors"
                        >
                          {ticket.subject}
                        </Link>
                      </td>
                      <td className="px-4 py-3 text-sm text-neutral-400">
                        {ticket.user_id.includes("@") ? ticket.user_id : ticket.user_id.slice(0, 8) + "..."}
                      </td>
                      <td className="px-4 py-3 text-sm text-neutral-400 capitalize">
                        {ticket.category.replace(/-/g, " ")}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize ${priorityColor(ticket.priority)}`}
                        >
                          {ticket.priority}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColor(ticket.status)}`}
                        >
                          {ticket.status.replace(/-/g, " ")}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-neutral-500">
                        {new Date(ticket.created_at).toLocaleDateString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-between text-sm text-neutral-500">
                <span>
                  Showing {offset + 1}–{Math.min(offset + limit, total)} of{" "}
                  {total}
                </span>
                <div className="flex gap-2">
                  <button
                    disabled={currentPage <= 1}
                    onClick={() => setOffset(offset - limit)}
                    className="rounded-md border border-border px-3 py-1 text-neutral-400 hover:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Previous
                  </button>
                  <button
                    disabled={currentPage >= totalPages}
                    onClick={() => setOffset(offset + limit)}
                    className="rounded-md border border-border px-3 py-1 text-neutral-400 hover:text-white disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </Shell>
  );
}
