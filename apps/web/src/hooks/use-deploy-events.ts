"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getAccessToken } from "@/lib/api";

/**
 * Shape of events received from the SSE deployment event stream.
 */
export interface DeployEventData {
  type: string;
  app_id: string;
  app_name: string;
  deployment_id: string;
  status: string;
  image?: string;
  message?: string;
  timestamp: string;
}

/**
 * Hook for subscribing to real-time deployment events via SSE.
 *
 * Connects to GET /api/v1/events (EventSource).
 * Automatically reconnects on connection loss.
 * Returns connection status + latest event.
 */
export function useDeployEvents(
  onEvent?: (event: DeployEventData) => void
): {
  connected: boolean;
  lastEvent: DeployEventData | null;
} {
  const [connected, setConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<DeployEventData | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const connect = useCallback(() => {
    if (typeof window === "undefined") return;

    const token = getAccessToken();
    if (!token) return;

    const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    const url = `${apiBase}/api/v1/events?token=${token}`;

    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => setConnected(true);

    es.addEventListener("deploy", (e) => {
      try {
        const data = JSON.parse(e.data) as DeployEventData;
        setLastEvent(data);
        if (onEventRef.current) onEventRef.current(data);
      } catch {
        // Ignore non-JSON events
      }
    });

    es.onerror = () => {
      setConnected(false);
      es.close();
      eventSourceRef.current = null;

      // Reconnect after 5 seconds
      reconnectTimerRef.current = setTimeout(() => {
        connect();
      }, 5000);
    };
  }, []);

  useEffect(() => {
    // Only connect in non-demo mode
    const isDemo = process.env.NEXT_PUBLIC_DEMO_MODE === "true";
    if (isDemo) return;

    connect();

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };
  }, [connect]);

  return { connected, lastEvent };
}
