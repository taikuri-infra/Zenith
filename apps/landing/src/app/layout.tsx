import type { Metadata } from "next";
import { Header } from "@/components/header";
import { Footer } from "@/components/footer";
import "./globals.css";

export const metadata: Metadata = {
  title: "Zenith - Your Own Cloud. Zero Complexity.",
  description:
    "100% free, open-source Kubernetes PaaS on Hetzner Cloud. Deploy apps, databases, auth, storage, and monitoring with a single command.",
  keywords: [
    "kubernetes",
    "paas",
    "open source",
    "hetzner",
    "cloud",
    "deployment",
    "self-hosted",
    "devops",
  ],
  openGraph: {
    title: "Zenith - Your Own Cloud. Zero Complexity.",
    description:
      "100% free, open-source Kubernetes PaaS on Hetzner Cloud. One command to deploy everything.",
    url: "https://freezenith.com",
    siteName: "Zenith",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Zenith - Your Own Cloud. Zero Complexity.",
    description:
      "100% free, open-source Kubernetes PaaS on Hetzner Cloud. One command to deploy everything.",
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
        <Header />
        <main>{children}</main>
        <Footer />
      </body>
    </html>
  );
}
