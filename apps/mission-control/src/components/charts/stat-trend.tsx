"use client";

import {
  ResponsiveContainer,
  AreaChart,
  Area,
} from "recharts";

interface StatTrendProps {
  label: string;
  value: string | number;
  sub?: string;
  trend?: number;
  data?: { value: number }[];
  alert?: boolean;
}

export function StatTrend({ label, value, sub, trend, data, alert }: StatTrendProps) {
  return (
    <div className="rounded-lg border border-border bg-surface-100 p-4">
      <div className="flex items-center justify-between">
        <span className="text-xs text-neutral-500">{label}</span>
        {trend !== undefined && (
          <span
            className={`text-xs font-medium ${
              trend >= 0 ? "text-emerald-400" : "text-red-400"
            }`}
          >
            {trend >= 0 ? "+" : ""}
            {trend.toFixed(1)}%
          </span>
        )}
      </div>
      <div className="mt-1 flex items-end justify-between">
        <div>
          <span
            className={`text-2xl font-semibold ${
              alert ? "text-amber-400" : "text-white"
            }`}
          >
            {value}
          </span>
          {sub && (
            <span className="ml-2 text-xs text-neutral-500">{sub}</span>
          )}
        </div>
        {data && data.length > 1 && (
          <div className="h-8 w-20">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={data}>
                <Area
                  type="monotone"
                  dataKey="value"
                  stroke={alert ? "#fbbf24" : "#6366f1"}
                  fill={alert ? "#fbbf2420" : "#6366f120"}
                  strokeWidth={1.5}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>
    </div>
  );
}
