import type { Metadata } from "next";
import { Header } from "@/components/header";
import { Footer } from "@/components/footer";
import "./globals.css";

export const metadata: Metadata = {
  title: "Zenith — Cloud Platform for Developers",
  description:
    "Deploy apps, databases, and APIs on Zenith Cloud in seconds — or self-host on your own infrastructure. Free tier available. Open source, MIT licensed.",
  keywords: [
    "cloud platform",
    "paas",
    "deployment",
    "kubernetes",
    "open source",
    "self-hosted",
    "hetzner",
    "devops",
    "infrastructure",
    "docker",
    "containers",
    "microservices",
    "saas",
    "developer tools",
  ],
  authors: [{ name: "DoTech", url: "https://dotech.com" }],
  openGraph: {
    title: "Zenith — Cloud Platform for Developers",
    description:
      "Deploy apps, databases, and APIs on Zenith Cloud in seconds — or self-host the open-source PaaS on your own infrastructure.",
    url: "https://freezenith.com",
    siteName: "Zenith",
    type: "website",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "Zenith — Cloud Platform for Developers",
    description:
      "Deploy on Zenith Cloud or self-host. Free tier, open source, MIT licensed.",
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
