"use client";

import { Shell } from "@/components/shell";
import { EmptyState } from "@/components/empty-state";
import { ListOrdered } from "lucide-react";

const queueEngines = [
  { name: "RabbitMQ", description: "AMQP message broker with routing, queuing, and pub/sub", icon: "R" },
  { name: "NATS", description: "Cloud-native messaging with JetStream persistence", icon: "N" },
];

export default function QueuesPage() {
  return (
    <Shell>
      <div className="space-y-6">
        <div>
          <h1 className="text-lg font-semibold text-white">Queues</h1>
          <p className="text-sm text-neutral-500">
            Managed message queues for async workloads
          </p>
        </div>

        <EmptyState
          title="Coming Soon"
          description="Managed message queues are on the roadmap. Stay tuned for RabbitMQ and NATS support."
        />

        <div className="grid gap-4 sm:grid-cols-2">
          {queueEngines.map((engine) => (
            <div
              key={engine.name}
              className="rounded-xl border border-border bg-surface-100 p-5 opacity-50"
            >
              <div className="flex items-center gap-3 mb-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-surface-300 text-sm font-bold text-neutral-400">
                  {engine.icon}
                </div>
                <div>
                  <h3 className="text-sm font-medium text-white">{engine.name}</h3>
                  <span className="rounded bg-neutral-500/20 px-1.5 py-0.5 text-[9px] text-neutral-500">
                    Coming Soon
                  </span>
                </div>
              </div>
              <p className="text-xs text-neutral-500">{engine.description}</p>
              <button
                disabled
                className="mt-4 w-full rounded-lg border border-border bg-surface-200 py-2 text-xs text-neutral-500 cursor-not-allowed"
              >
                Create Queue
              </button>
            </div>
          ))}
        </div>
      </div>
    </Shell>
  );
}
