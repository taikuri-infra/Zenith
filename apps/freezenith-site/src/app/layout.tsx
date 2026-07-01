import type { Metadata } from "next";
import { Header } from "@/components/header";
import { Footer } from "@/components/footer";
import { site } from "@/lib/site";
import "./globals.css";

export const metadata: Metadata = {
  title: "FreeZenith — Source-available private cloud you self-host",
  description: site.description,
  keywords: [
    "internal developer platform",
    "idp",
    "private cloud",
    "self-hosted",
    "source available",
    "kubernetes",
    "k3s",
    "backstage",
    "cilium",
    "kyverno",
    "velero",
    "victoriametrics",
    "hetzner",
    "on-premises",
    "platform engineering",
    "devops",
  ],
  authors: [{ name: "FreeZenith" }],
  openGraph: {
    title: "FreeZenith — Source-available private cloud you self-host",
    description: site.description,
    url: "https://freezenith.com",
    siteName: "FreeZenith",
    type: "website",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "FreeZenith — Source-available private cloud you self-host",
    description: site.description,
  },
  robots: { index: true, follow: true },
  metadataBase: new URL("https://freezenith.com"),
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark" suppressHydrationWarning>
      <body className="min-h-screen bg-surface font-sans text-white antialiased">
        <Header />
        <main>{children}</main>
        <Footer />
      </body>
    </html>
  );
}
