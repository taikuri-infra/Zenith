"use client";

import { useCallback, useEffect, useRef, useState } from "react";

export interface LogEntry {
  timestamp: string;
  level: "info" | "warn" | "error" | "build" | "deploy";
  message: string;
}

interface LogHistoryResponse {
  items: LogEntry[];
  total: number;
}

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

function getAuthHeader(): Record<string, string> {
  if (typeof window === "undefined") return {};
  const token = localStorage.getItem("zenith_access_token");
  return token ? { Authorization: `Bearer ${token}` } : {};
}

/**
 * Hook for streaming deployment logs via SSE.
 *
 * - On mount: fetches existing log history (GET .../logs/history)
 * - Then opens EventSource for live updates (GET .../logs)
 * - Closes the stream when the parent component unmounts or when
 *   the server fires `event: done`.
 *
 * In demo mode (NEXT_PUBLIC_DEMO_MODE=true) it returns a set of
 * hardcoded sample log lines instead.
 */
export function useDeployLogs(
  appId: string | null,
  deploymentId: string | null
): {
  entries: LogEntry[];
  streaming: boolean;
} {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [streaming, setStreaming] = useState(false);
  const esRef = useRef<EventSource | null>(null);

  const append = useCallback((entry: LogEntry) => {
    setEntries((prev) => [...prev, entry]);
  }, []);

  useEffect(() => {
    if (!appId || !deploymentId) return;

    // Demo mode: return static sample logs
    if (process.env.NEXT_PUBLIC_DEMO_MODE === "true") {
      const demo: LogEntry[] = [
        {
          timestamp: new Date(Date.now() - 12000).toISOString(),
          level: "info",
          message: "Cloning repository https://github.com/acme/my-next-app...",
        },
        {
          timestamp: new Date(Date.now() - 10000).toISOString(),
          level: "info",
          message: "Detected framework: Next.js",
        },
        {
          timestamp: new Date(Date.now() - 9000).toISOString(),
          level: "build",
          message: "Generating multi-stage Dockerfile...",
        },
        {
          timestamp: new Date(Date.now() - 8000).toISOString(),
          level: "build",
          message: "Building image zenith/my-next-app:abc12345...",
        },
        {
          timestamp: new Date(Date.now() - 4000).toISOString(),
          level: "build",
          message: "Build complete: zenith/my-next-app:abc12345",
        },
        {
          timestamp: new Date(Date.now() - 3000).toISOString(),
          level: "deploy",
          message: "Deploying to Kubernetes...",
        },
        {
          timestamp: new Date(Date.now() - 2000).toISOString(),
          level: "deploy",
          message:
            "✓ Deployed successfully — my-next-app is live",
        },
      ];
      setEntries(demo);
      return;
    }

    let cancelled = false;

    async function init() {
      // 1. Fetch history snapshot
      try {
        const res = await fetch(
          `${API_BASE}/api/v1/apps/${appId}/deployments/${deploymentId}/logs/history`,
          { headers: getAuthHeader() }
        );
        if (res.ok) {
          const data: LogHistoryResponse = await res.json();
          if (!cancelled) {
            setEntries(data.items ?? []);
          }
        }
      } catch {
        // Non-fatal — we can still open the live stream
      }

      if (cancelled) return;

      // 2. Open SSE stream for live entries
      // EventSource doesn't support custom headers, so we pass the
      // token as a query param when needed — for now the token is
      // read from the Authorization header in Fiber via cookie or
      // the server falls back; in dev mode CORS is open.
      const url = `${API_BASE}/api/v1/apps/${appId}/deployments/${deploymentId}/logs`;
      const es = new EventSource(url);
      esRef.current = es;
      setStreaming(true);

      es.onmessage = (event) => {
        try {
          const entry: LogEntry = JSON.parse(event.data);
          if (!cancelled) append(entry);
        } catch {
          // keep-alive or malformed — ignore
        }
      };

      // server fires `event: done` when the deployment finishes
      es.addEventListener("done", () => {
        setStreaming(false);
        es.close();
        esRef.current = null;
      });

      es.onerror = () => {
        setStreaming(false);
        es.close();
        esRef.current = null;
      };
    }

    init();

    return () => {
      cancelled = true;
      if (esRef.current) {
        esRef.current.close();
        esRef.current = null;
      }
      setStreaming(false);
    };
  }, [appId, deploymentId, append]);

  return { entries, streaming };
}
