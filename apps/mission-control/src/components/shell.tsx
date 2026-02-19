"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Sidebar } from "./sidebar";
import { Header } from "./header";
import { DemoBanner } from "./demo-banner";
import { useAuth } from "@/hooks/use-auth";
import { isDemoMode } from "@/lib/get-api";

export function Shell({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth();
  const router = useRouter();
  const demo = isDemoMode();

  useEffect(() => {
    if (!demo && !loading && !isAuthenticated) {
      router.replace("/login");
    }
  }, [demo, loading, isAuthenticated, router]);

  if (!demo && loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-neutral-950">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-neutral-700 border-t-emerald-500" />
      </div>
    );
  }

  if (!demo && !isAuthenticated) {
    return null;
  }

  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col pl-60">
        <DemoBanner />
        <Header />
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}
