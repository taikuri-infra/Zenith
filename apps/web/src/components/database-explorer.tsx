"use client";

import { useState, useEffect, useRef } from "react";
import { getApi } from "@/lib/get-api";
import { Play, Square, Shield, ShieldOff, Loader2, AlertTriangle } from "lucide-react";

interface DatabaseExplorerProps {
  dbId: string;
  engine: string;
}

export function DatabaseExplorer({ dbId, engine }: DatabaseExplorerProps) {
  const { standaloneDatabases } = getApi();
  const [loading, setLoading] = useState(false);
  const [checking, setChecking] = useState(true);
  const [session, setSession] = useState<{
    url: string;
    status: string;
    readonly: boolean;
  } | null>(null);
  const [readOnly, setReadOnly] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Check for existing session on mount
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const status = await standaloneDatabases.explorerStatus(dbId);
        if (!cancelled && status.active && status.url) {
          setSession({
            url: status.url,
            status: status.status || "running",
            readonly: status.readonly ?? true,
          });
        }
      } catch {
        // No session — that's fine
      } finally {
        if (!cancelled) setChecking(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [dbId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  if (engine !== "postgresql") {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <AlertTriangle className="h-10 w-10 text-neutral-500 mb-3" />
        <h3 className="text-sm font-medium text-white mb-1">Not Supported</h3>
        <p className="text-xs text-neutral-500 max-w-sm">
          Database Explorer is only available for PostgreSQL databases. This
          database uses {engine}.
        </p>
      </div>
    );
  }

  if (checking) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="h-5 w-5 animate-spin text-neutral-500" />
      </div>
    );
  }

  const handleStart = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await standaloneDatabases.startExplorer(dbId, readOnly);
      if (result.status === "starting") {
        // Poll for readiness
        setSession({ ...result, status: "starting" });
        pollRef.current = setInterval(async () => {
          try {
            const status = await standaloneDatabases.explorerStatus(dbId);
            if (status.active && status.status === "running") {
              setSession({
                url: status.url!,
                status: "running",
                readonly: status.readonly ?? true,
              });
              if (pollRef.current) clearInterval(pollRef.current);
            }
          } catch {
            // Keep polling
          }
        }, 3000);
      } else {
        setSession(result);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to start explorer");
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    setLoading(true);
    try {
      await standaloneDatabases.stopExplorer(dbId);
      setSession(null);
      if (pollRef.current) clearInterval(pollRef.current);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to stop explorer");
    } finally {
      setLoading(false);
    }
  };

  // Active session — show iframe
  if (session) {
    return (
      <div className="flex flex-col h-[calc(100vh-280px)] min-h-[500px]">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            {session.status === "starting" ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin text-amber-400" />
                <span className="text-xs text-amber-400">Starting explorer...</span>
              </>
            ) : (
              <>
                <div className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse" />
                <span className="text-xs text-neutral-400">Explorer running</span>
              </>
            )}
            {session.readonly ? (
              <span className="flex items-center gap-1 rounded-full bg-blue-500/15 px-2 py-0.5 text-[10px] font-medium text-blue-400">
                <Shield className="h-3 w-3" /> Read-only
              </span>
            ) : (
              <span className="flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-[10px] font-medium text-amber-400">
                <ShieldOff className="h-3 w-3" /> Read-write
              </span>
            )}
          </div>
          <button
            onClick={handleStop}
            disabled={loading}
            className="flex items-center gap-1.5 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-1.5 text-xs font-medium text-red-400 hover:bg-red-500/20 transition-colors disabled:opacity-50"
          >
            <Square className="h-3 w-3" />
            Stop Explorer
          </button>
        </div>
        {session.status === "running" ? (
          <iframe
            src={session.url}
            className="flex-1 w-full rounded-lg border border-border bg-white"
            title="Database Explorer"
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
          />
        ) : (
          <div className="flex-1 flex items-center justify-center rounded-lg border border-border bg-surface-100">
            <div className="text-center">
              <Loader2 className="h-8 w-8 animate-spin text-accent-400 mx-auto mb-3" />
              <p className="text-sm text-neutral-400">Starting pgweb pod...</p>
              <p className="text-xs text-neutral-500 mt-1">This usually takes 10-20 seconds</p>
            </div>
          </div>
        )}
      </div>
    );
  }

  // No session — show start button
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-accent-500/10">
        <Play className="h-7 w-7 text-accent-400" />
      </div>
      <h3 className="text-sm font-medium text-white mb-1">Database Explorer</h3>
      <p className="text-xs text-neutral-500 max-w-sm mb-6">
        Browse tables, inspect data, and run queries directly in your browser.
        Powered by pgweb. Sessions auto-expire after 30 minutes.
      </p>

      {error && (
        <div className="mb-4 rounded-lg border border-red-500/30 bg-red-500/5 px-4 py-2 text-xs text-red-400">
          {error}
        </div>
      )}

      <div className="flex items-center gap-3 mb-6">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={readOnly}
            onChange={(e) => setReadOnly(e.target.checked)}
            className="rounded border-border bg-surface-200 text-accent-500 focus:ring-accent-500"
          />
          <span className="text-xs text-neutral-400">Read-only mode</span>
        </label>
      </div>

      <button
        onClick={handleStart}
        disabled={loading}
        className="flex items-center gap-2 rounded-lg bg-accent-500 px-5 py-2.5 text-sm font-medium text-white hover:bg-accent-600 transition-colors disabled:opacity-50"
      >
        {loading ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <Play className="h-4 w-4" />
        )}
        {loading ? "Starting..." : "Start Explorer"}
      </button>
    </div>
  );
}
