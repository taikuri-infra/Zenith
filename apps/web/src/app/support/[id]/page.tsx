"use client";

import { Shell } from "@/components/shell";
import { ErrorState } from "@/components/error-state";
import { useApi } from "@/hooks/use-api";
import { type SupportMessage } from "@/lib/api";
import { getApi } from "@/lib/get-api";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";

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

export default function SupportTicketPage() {
  const params = useParams();
  const ticketId = params.id as string;
  const { support } = getApi();

  const { data, loading, error, refetch } = useApi(
    () => support.get(ticketId),
    [ticketId]
  );

  const [replyBody, setReplyBody] = useState("");
  const [sending, setSending] = useState(false);

  const ticket = data?.ticket;
  const messages: SupportMessage[] = data?.messages ?? [];

  const isClosed =
    ticket?.status === "closed" || ticket?.status === "resolved";

  const handleReply = async () => {
    if (!replyBody.trim() || sending) return;
    setSending(true);
    try {
      await support.reply(ticketId, replyBody.trim());
      setReplyBody("");
      refetch();
    } catch {
      // error handled by UI
    } finally {
      setSending(false);
    }
  };

  if (loading) {
    return (
      <Shell>
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
      </Shell>
    );
  }

  if (error || !ticket) {
    return (
      <Shell>
        <ErrorState
          message={error || "Ticket not found"}
          onRetry={refetch}
        />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="space-y-6">
        {/* Back link + header */}
        <div>
          <Link
            href="/support"
            className="text-sm text-neutral-500 hover:text-neutral-300 transition-colors"
          >
            &larr; Back to tickets
          </Link>
        </div>

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">
              {ticket.subject}
            </h1>
            <div className="mt-2 flex items-center gap-3 text-sm">
              <span
                className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColor(ticket.status)}`}
              >
                {ticket.status.replace(/-/g, " ")}
              </span>
              <span
                className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${priorityColor(ticket.priority)}`}
              >
                {ticket.priority}
              </span>
              <span className="text-neutral-500 capitalize">
                {ticket.category.replace(/-/g, " ")}
              </span>
              <span className="text-neutral-600">
                {new Date(ticket.created_at).toLocaleDateString()}
              </span>
            </div>
          </div>
        </div>

        {/* Message thread */}
        <div className="space-y-4">
          {messages.map((msg) => (
            <div
              key={msg.id}
              className={`rounded-xl border p-4 ${
                msg.sender_role === "admin"
                  ? "border-accent-500/30 bg-accent-500/5 ml-8"
                  : "border-border bg-surface-100 mr-8"
              }`}
            >
              <div className="flex items-center gap-2 mb-2">
                <span
                  className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                    msg.sender_role === "admin"
                      ? "bg-accent-500/15 text-accent-400"
                      : "bg-neutral-500/15 text-neutral-400"
                  }`}
                >
                  {msg.sender_role === "admin" ? "Support Team" : "You"}
                </span>
                <span className="text-xs text-neutral-600">
                  {new Date(msg.created_at).toLocaleString()}
                </span>
              </div>
              <p className="text-sm text-neutral-300 whitespace-pre-wrap">
                {msg.body}
              </p>
            </div>
          ))}
        </div>

        {/* Reply box */}
        {isClosed ? (
          <div className="rounded-lg border border-border bg-surface-100 p-4 text-center text-sm text-neutral-500">
            This ticket has been {ticket.status}. If you need further help,
            please create a new ticket.
          </div>
        ) : (
          <div className="rounded-xl border border-border bg-surface-100 p-4">
            <textarea
              value={replyBody}
              onChange={(e) => setReplyBody(e.target.value)}
              placeholder="Write your reply..."
              rows={3}
              className="w-full rounded-lg border border-border bg-surface-50 px-3 py-2 text-sm text-white placeholder-neutral-500 outline-none focus:border-accent-500 focus:ring-1 focus:ring-accent-500 resize-none"
            />
            <div className="mt-3 flex justify-end">
              <button
                onClick={handleReply}
                disabled={sending || !replyBody.trim()}
                className="rounded-lg bg-accent-500 hover:bg-accent-600 px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {sending ? "Sending..." : "Reply"}
              </button>
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}
