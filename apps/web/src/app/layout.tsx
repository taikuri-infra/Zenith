import type { Metadata } from "next";
import Script from "next/script";
import { ToastProvider } from "@/components/toast";
import "./globals.css";

export const metadata: Metadata = {
  title: "Zenith - Open-Source PaaS for Kubernetes",
  description: "Deploy apps, databases, and infrastructure on Kubernetes without the complexity.",
  icons: {
    icon: "/favicon.svg",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-surface font-sans">
        <Script src="/_next/static/env.js" strategy="beforeInteractive" />
        <ToastProvider>{children}</ToastProvider>
      </body>
    </html>
  );
}
