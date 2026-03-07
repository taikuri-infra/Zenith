"use client";

import { Sidebar } from "./sidebar";
import { Header } from "./header";
import { DemoBanner } from "./demo-banner";
import { ProjectContext, useProjectState } from "@/hooks/use-project";

export function Shell({ children }: { children: React.ReactNode }) {
  const projectState = useProjectState();

  return (
    <ProjectContext.Provider value={projectState}>
      <div className="flex min-h-screen">
        <Sidebar />
        <div className="flex flex-1 flex-col pl-56">
          <DemoBanner />
          <Header />
          <main className="flex-1 p-6">{children}</main>
        </div>
      </div>
    </ProjectContext.Provider>
  );
}
