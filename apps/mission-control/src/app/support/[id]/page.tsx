"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Shell } from "@/components/shell";
import { getApi } from "@/lib/get-api";
import type { SupportTicket, SupportMessage } from "@/lib/api";

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

function priorityBadge(priority: string): string {
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

const statusOptions = [
  "open",
  "in-progress",
  "waiting-on-customer",
  "resolved",
  "closed",
];

export default function AdminSupportTicketPage() {
  const apiClient = getApi();
  const params = useParams();
  const ticketId = params.id as string;

  const [ticket, setTicket] = useState<SupportTicket | null>(null);
  const [messages, setMessages] = useState<SupportMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [replyBody, setReplyBody] = useState("");
  const [sending, setSending] = useState(false);
  const [updatingStatus, setUpdatingStatus] = useState(false);

  const fetchTicket = async () => {
    setLoading(true);
    setError("");
    try {
      const res = await apiClient.support.get(ticketId);
      setTicket(res.ticket);
      setMessages(res.messages ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load ticket");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTicket();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ticketId]);

  const handleReply = async () => {
    if (!replyBody.trim() || sending) return;
    setSending(true);
    try {
      await apiClient.support.reply(ticketId, replyBody.trim());
      setReplyBody("");
      fetchTicket();
    } catch {
      // handled
    } finally {
      setSending(false);
    }
  };

  const handleStatusChange = async (status: string) => {
    setUpdatingStatus(true);
    try {
      await apiClient.support.updateStatus(ticketId, status);
      fetchTicket();
    } catch {
      // handled
    } finally {
      setUpdatingStatus(false);
    }
  };

  return (
    <Shell>
      {loading ? (
        <div className="space-y-4">
          <div className="h-6 w-48 animate-pulse rounded bg-surface-300" />
          <div className="h-4 w-32 animate-pulse rounded bg-surface-300" />
          <div className="space-y-3 mt-8">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="h-20 animate-pulse rounded-lg bg-surface-200"
              />
            ))}
          </div>
        </div>
      ) : error || !ticket ? (
        <div className="rounded-xl border border-border bg-surface-100 py-12 text-center">
          <p className="text-sm text-red-400">{error || "Ticket not found"}</p>
          <div className="mt-4 flex items-center justify-center gap-3">
            <button
              onClick={fetchTicket}
              className="text-sm text-accent-400 hover:underline"
            >
              Retry
            </button>
            <Link
              href="/support"
              className="text-sm text-neutral-400 hover:text-white transition-colors"
            >
              Back to tickets
            </Link>
          </div>
        </div>
      ) : (
        <div className="space-y-6">
          {/* Back */}
          <Link
            href="/support"
            className="inline-flex items-center gap-1.5 text-sm text-neutral-500 hover:text-neutral-300 transition-colors"
          >
            &larr; Back to tickets
          </Link>

          {/* Header */}
          <div className="rounded-xl border border-border bg-surface-100 p-5">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <h1 className="text-lg font-semibold text-white">
                  {ticket.subject}
                </h1>
                <div className="mt-3 flex flex-wrap items-center gap-3">
                  <span
                    className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${statusColor(ticket.status)}`}
                  >
                    {ticket.status.replace(/-/g, " ")}
                  </span>
                  <span
                    className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${priorityBadge(ticket.priority)}`}
                  >
                    {ticket.priority}
                  </span>
                  <span className="text-xs text-neutral-500 capitalize">
                    {ticket.category.replace(/-/g, " ")}
                  </span>
                  <span className="text-neutral-600">|</span>
                  <span className="text-xs text-neutral-500">
                    User: {ticket.user_id.includes("@") ? ticket.user_id : ticket.user_id.slice(0, 12) + "..."}
                  </span>
                  <span className="text-xs text-neutral-600">
                    Created {new Date(ticket.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>

              {/* Status dropdown */}
              <select
                value={ticket.status}
                onChange={(e) => handleStatusChange(e.target.value)}
                disabled={updatingStatus}
                className="rounded-lg border border-border bg-surface-200 px-3 py-1.5 text-sm text-white outline-none focus:border-accent-600"
              >
                {statusOptions.map((s) => (
                  <option key={s} value={s}>
                    {s.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase())}
                  </option>
                ))}
              </select>
            </div>

            {/* Quick status buttons */}
            <div className="mt-4 flex gap-2 border-t border-border pt-4">
              {[
                { label: "Mark Resolved", status: "resolved", color: "bg-green-600 hover:bg-green-700" },
                { label: "Set Waiting", status: "waiting-on-customer", color: "bg-orange-600 hover:bg-orange-700" },
                { label: "Close", status: "closed", color: "bg-neutral-600 hover:bg-neutral-700" },
              ].map((btn) => (
                <button
                  key={btn.status}
                  onClick={() => handleStatusChange(btn.status)}
                  disabled={updatingStatus || ticket.status === btn.status}
                  className={`rounded-md px-3 py-1.5 text-xs font-medium text-white transition-colors disabled:opacity-50 ${btn.color}`}
                >
                  {btn.label}
                </button>
              ))}
            </div>
          </div>

          {/* Message thread */}
          <div className="space-y-4">
            <h2 className="text-sm font-medium text-neutral-400">Conversation</h2>
            {messages.length === 0 ? (
              <div className="rounded-xl border border-border bg-surface-100 py-8 text-center text-sm text-neutral-500">
                No messages yet
              </div>
            ) : (
              messages.map((msg) => (
                <div
                  key={msg.id}
                  className={`rounded-xl border p-4 ${
                    msg.sender_role === "admin"
                      ? "border-accent-600/30 bg-accent-600/5 ml-8"
                      : "border-border bg-surface-100 mr-8"
                  }`}
                >
                  <div className="flex items-center gap-2 mb-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                        msg.sender_role === "admin"
                          ? "bg-accent-600/15 text-accent-400"
                          : "bg-neutral-500/15 text-neutral-400"
                      }`}
                    >
                      {msg.sender_role === "admin" ? "Admin" : "Customer"}
                    </span>
                    <span className="text-xs text-neutral-500">
                      {new Date(msg.created_at).toLocaleString()}
                    </span>
                  </div>
                  <p className="text-sm text-neutral-300 whitespace-pre-wrap">
                    {msg.body}
                  </p>
                </div>
              ))
            )}
          </div>

          {/* Reply box */}
          <div className="rounded-xl border border-border bg-surface-100 p-4">
            <h3 className="text-sm font-medium text-neutral-400 mb-3">Reply</h3>
            <textarea
              value={replyBody}
              onChange={(e) => setReplyBody(e.target.value)}
              placeholder="Write your reply..."
              rows={4}
              className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-600 focus:ring-1 focus:ring-accent-600 resize-none"
            />
            <div className="mt-3 flex justify-end">
              <button
                onClick={handleReply}
                disabled={sending || !replyBody.trim()}
                className="rounded-lg bg-accent-600 hover:bg-accent-700 px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {sending ? "Sending..." : "Reply"}
              </button>
            </div>
          </div>
        </div>
      )}
    </Shell>
  );
}
