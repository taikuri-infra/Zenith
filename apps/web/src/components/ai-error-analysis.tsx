"use client";

import { useState } from "react";
import { getApi } from "@/lib/get-api";
import type { ErrorAnalysisResult, AIUsageInfo } from "@/lib/api";
import { useToast } from "@/components/toast";
import { Zap, AlertTriangle, CheckCircle, Info, Loader2 } from "lucide-react";

interface AIErrorAnalysisProps {
  appId: string;
}

export function AIErrorAnalysis({ appId }: AIErrorAnalysisProps) {
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ErrorAnalysisResult | null>(null);
  const [usage, setUsage] = useState<AIUsageInfo | null>(null);

  const analyze = async () => {
    setLoading(true);
    try {
      const api = getApi();
      const [analysis, usageInfo] = await Promise.all([
        api.ai.analyzeError(appId),
        api.ai.getUsage(),
      ]);
      setResult(analysis);
      setUsage(usageInfo);
    } catch (err) {
      toast("error", err instanceof Error ? err.message : "AI analysis failed");
    } finally {
      setLoading(false);
    }
  };

  const confidenceColor = (c: string) => {
    switch (c) {
      case "high": return "text-green-400";
      case "medium": return "text-yellow-400";
      case "low": return "text-red-400";
      default: return "text-zinc-400";
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <button
          onClick={analyze}
          disabled={loading}
          className="flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
          {loading ? "Analyzing..." : "Why did this crash?"}
        </button>
        {usage && (
          <span className="text-xs text-zinc-500">
            {usage.monthly_used} / {usage.monthly_limit} AI analyses this month
          </span>
        )}
      </div>

      {result && (
        <div className="space-y-3 rounded-lg border border-zinc-700 bg-zinc-800/50 p-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-white">AI Error Analysis</h3>
            <span className={`text-xs font-medium ${confidenceColor(result.confidence)}`}>
              Confidence: {result.confidence}
            </span>
          </div>

          <div className="space-y-3">
            <div className="flex gap-2">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-400" />
              <div>
                <p className="text-xs font-medium text-zinc-400">Problem</p>
                <p className="text-sm text-zinc-200">{result.problem}</p>
              </div>
            </div>

            <div className="flex gap-2">
              <Info className="mt-0.5 h-4 w-4 shrink-0 text-blue-400" />
              <div>
                <p className="text-xs font-medium text-zinc-400">Root Cause</p>
                <p className="text-sm text-zinc-200">{result.cause}</p>
              </div>
            </div>

            <div className="flex gap-2">
              <CheckCircle className="mt-0.5 h-4 w-4 shrink-0 text-green-400" />
              <div>
                <p className="text-xs font-medium text-zinc-400">Suggested Fix</p>
                <pre className="whitespace-pre-wrap text-sm text-zinc-200">{result.fix}</pre>
              </div>
            </div>
          </div>

          <p className="mt-2 text-xs text-zinc-500 italic">{result.pii_disclaimer}</p>
        </div>
      )}
    </div>
  );
}
