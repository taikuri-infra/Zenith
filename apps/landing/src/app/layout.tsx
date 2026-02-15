import type { Metadata } from "next";
import { Header } from "@/components/header";
import { Footer } from "@/components/footer";
import "./globals.css";

export const metadata: Metadata = {
  title: "Zenith - Your Own Cloud Platform. 10x Cheaper.",
  description:
    "One command turns Hetzner Cloud into your own platform. Apps, databases, auth, storage, gateway, monitoring. 100% free and open source. 10x cheaper than AWS.",
  keywords: [
    "kubernetes",
    "paas",
    "open source",
    "hetzner",
    "cloud platform",
    "deployment",
    "self-hosted",
    "devops",
    "infrastructure",
    "docker",
    "containers",
    "microservices",
  ],
  authors: [{ name: "DoTech", url: "https://dotech.com" }],
  openGraph: {
    title: "Zenith - Your Own Cloud Platform. 10x Cheaper.",
    description:
      "One zen install command turns Hetzner Cloud into your own platform. Apps, databases, auth, storage, gateway, monitoring. 100% free and open source.",
    url: "https://freezenith.com",
    siteName: "Zenith",
    type: "website",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "Zenith - Your Own Cloud Platform. 10x Cheaper.",
    description:
      "One command turns Hetzner Cloud into your own platform. 100% free, open source. 10x cheaper than AWS.",
    creator: "@freezenith",
  },
  robots: {
    index: true,
    follow: true,
  },
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
