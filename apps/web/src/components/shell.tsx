"use client";

import { Sidebar } from "./sidebar";
import { Header } from "./header";
import { DemoBanner } from "./demo-banner";

export function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col pl-56">
        <DemoBanner />
        <Header />
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}
