"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { connectWebSocket, WebSocketEvent } from "@/lib/api";

/**
 * Hook for managing WebSocket connections for real-time updates.
 */
export function useWebSocket(
  projectId: string | null,
  onEvent?: (event: WebSocketEvent) => void
): {
  connected: boolean;
  lastEvent: WebSocketEvent | null;
} {
  const [connected, setConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<WebSocketEvent | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const handleMessage = useCallback(
    (event: WebSocketEvent) => {
      setLastEvent(event);
      if (onEvent) onEvent(event);
    },
    [onEvent]
  );

  useEffect(() => {
    if (!projectId) return;

    const ws = connectWebSocket(
      projectId,
      handleMessage,
      () => setConnected(false)
    );

    if (ws) {
      wsRef.current = ws;

      ws.onopen = () => setConnected(true);
      ws.onclose = () => setConnected(false);

      return () => {
        ws.close();
        wsRef.current = null;
      };
    }
  }, [projectId, handleMessage]);

  return { connected, lastEvent };
}
