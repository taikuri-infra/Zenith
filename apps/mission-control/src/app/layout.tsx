import type { Metadata } from "next";
import Script from "next/script";
import "./globals.css";

export const metadata: Metadata = {
  title: "Mission Control | Zenith",
  description: "Platform management for Zenith",
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
        {children}
      </body>
    </html>
  );
}
