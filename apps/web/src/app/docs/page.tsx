"use client";

import { Shell } from "@/components/shell";
import {
  Rocket,
  Database,
  Shield,
  HardDrive,
  GitBranch,
  Terminal,
  Globe,
  Copy,
  Check,
} from "lucide-react";
import { useState } from "react";

function CodeBlock({ code, lang = "bash" }: { code: string; lang?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="group relative rounded-lg bg-surface-200 font-mono text-xs">
      <div className="flex items-center justify-between border-b border-border px-3 py-1.5">
        <span className="text-[10px] text-neutral-600 uppercase">{lang}</span>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 text-[10px] text-neutral-500 hover:text-white transition-colors"
        >
          {copied ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="overflow-x-auto p-3 text-neutral-300 leading-relaxed">{code}</pre>
    </div>
  );
}

function Step({
  number,
  title,
  icon: Icon,
  children,
}: {
  number: number;
  title: string;
  icon: React.ComponentType<{ className?: string }>;
  children: React.ReactNode;
}) {
  return (
    <div className="relative pl-10">
      <div className="absolute left-0 top-0 flex h-7 w-7 items-center justify-center rounded-full bg-accent-500/20 text-xs font-bold text-accent-400">
        {number}
      </div>
      <h3 className="flex items-center gap-2 text-sm font-medium text-white mb-3">
        <Icon className="h-4 w-4 text-accent-400" />
        {title}
      </h3>
      <div className="space-y-3 text-sm text-neutral-400">{children}</div>
    </div>
  );
}

export default function DocsPage() {
  return (
    <Shell>
      <div className="mx-auto max-w-3xl space-y-8">
        {/* Header */}
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Rocket className="h-5 w-5 text-accent-400" />
            <h1 className="text-lg font-semibold text-white">Deploy a Full-Stack App in 5 Minutes</h1>
          </div>
          <p className="text-sm text-neutral-500">
            Go from zero to a live app with a database, auth, and storage — all on Zenith.
          </p>
        </div>

        {/* Prerequisites */}
        <div className="rounded-lg border border-border bg-surface-100 p-4">
          <h2 className="mb-2 text-sm font-medium text-white">Prerequisites</h2>
          <ul className="space-y-1 text-xs text-neutral-400">
            <li>A Zenith account (sign up at freezenith.com)</li>
            <li>A GitHub repository with your app code</li>
            <li>Node.js, Go, Python, or any Dockerfile-based app</li>
          </ul>
        </div>

        {/* Steps */}
        <div className="space-y-8">
          <Step number={1} title="Create Your App" icon={GitBranch}>
            <p>
              Go to the <strong className="text-white">Apps</strong> page and click <strong className="text-white">New App</strong>.
              Provide your GitHub repo URL and branch.
            </p>
            <CodeBlock
              lang="api"
              code={`POST /api/v1/apps
{
  "name": "my-next-app",
  "repo_url": "https://github.com/you/my-next-app",
  "branch": "main"
}`}
            />
            <p>
              Zenith auto-detects your framework (Next.js, Go, Flask, etc.) and assigns a subdomain like{" "}
              <code className="rounded bg-surface-200 px-1.5 py-0.5 text-accent-400">my-next-app.freezenith.com</code>.
            </p>
          </Step>

          <Step number={2} title="Add a Database" icon={Database}>
            <p>
              Open your app, go to the <strong className="text-white">Databases</strong> tab, and click{" "}
              <strong className="text-white">Add Database</strong>. Choose PostgreSQL, MySQL, or Redis.
            </p>
            <CodeBlock
              lang="api"
              code={`POST /api/v1/apps/:appId/databases
{ "engine": "postgresql" }

# Returns:
{
  "id": "db-abc123",
  "name": "db-my-next",
  "engine": "postgresql",
  "host": "localhost",
  "port": 5432,
  "status": "ready"
}`}
            />
            <p>
              The <code className="rounded bg-surface-200 px-1.5 py-0.5 text-accent-400">DATABASE_URL</code> environment
              variable is automatically injected into your app. No manual config needed.
            </p>
          </Step>

          <Step number={3} title="Enable Auth" icon={Shield}>
            <p>
              Go to the <strong className="text-white">Auth</strong> tab and click{" "}
              <strong className="text-white">Enable Auth</strong>. Your app gets its own user table and JWT tokens.
            </p>
            <CodeBlock
              lang="javascript"
              code={`// Sign up a user in your app
const res = await fetch(
  "https://api.freezenith.com/api/v1/apps/YOUR_APP_ID/auth/signup",
  {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email: "user@example.com",
      password: "securepassword",
      name: "Jane Doe"
    })
  }
);

const { access_token } = await res.json();
// Use access_token for authenticated requests`}
            />
            <p>
              Each app gets isolated auth with its own JWT secret. Users are managed from the Auth tab in your app dashboard.
            </p>
          </Step>

          <Step number={4} title="Add Storage" icon={HardDrive}>
            <p>
              Go to the <strong className="text-white">Storage</strong> tab and create an S3-compatible bucket for file uploads.
            </p>
            <CodeBlock
              lang="api"
              code={`POST /api/v1/apps/:appId/storage
{ "name": "uploads", "access": "private" }

# Returns:
{
  "id": "bkt-xyz789",
  "name": "uploads",
  "endpoint": "https://uploads.s3.zenith.local",
  "status": "active"
}`}
            />
            <p>
              <code className="rounded bg-surface-200 px-1.5 py-0.5 text-accent-400">S3_ENDPOINT</code> and{" "}
              <code className="rounded bg-surface-200 px-1.5 py-0.5 text-accent-400">S3_BUCKET</code> are auto-injected.
              Use any S3-compatible SDK (like AWS SDK) to upload/download files.
            </p>
          </Step>

          <Step number={5} title="Deploy via Git Push" icon={Terminal}>
            <p>
              Set up the GitHub webhook to auto-deploy on push. Or trigger manual deploys from the dashboard.
            </p>
            <CodeBlock
              lang="bash"
              code={`# Push to your repo — Zenith builds and deploys automatically
git add .
git commit -m "feat: initial release"
git push origin main

# Your app is live at:
# https://my-next-app.freezenith.com`}
            />
            <p>
              Every push to your configured branch triggers a build. View build logs, deployment history, and rollback
              to any previous version from the Deployments tab.
            </p>
          </Step>

          <Step number={6} title="Go Live" icon={Globe}>
            <p>Your app is now running with:</p>
            <ul className="list-disc list-inside space-y-1 text-xs text-neutral-400 pl-2">
              <li>Auto-scaling containers with SSL</li>
              <li>Managed PostgreSQL with auto-injected connection string</li>
              <li>Built-in user authentication with JWT tokens</li>
              <li>S3-compatible object storage for file uploads</li>
              <li>Automatic database backups (Pro+ plans)</li>
              <li>One-click rollbacks to any previous deployment</li>
            </ul>
          </Step>
        </div>

        {/* API Reference quick links */}
        <div className="rounded-lg border border-border bg-surface-100 p-5">
          <h2 className="mb-3 text-sm font-medium text-white">API Quick Reference</h2>
          <div className="grid grid-cols-2 gap-3 text-xs">
            {[
              { label: "Apps", endpoints: ["POST /apps", "GET /apps", "DELETE /apps/:id"] },
              { label: "Databases", endpoints: ["POST /apps/:id/databases", "GET /databases", "DELETE /apps/:id/databases/:dbId"] },
              { label: "Auth", endpoints: ["POST /apps/:id/auth/enable", "POST /apps/:id/auth/signup", "POST /apps/:id/auth/login"] },
              { label: "Storage", endpoints: ["POST /apps/:id/storage", "GET /storage-buckets", "DELETE /apps/:id/storage/:bktId"] },
              { label: "Backups", endpoints: ["POST /apps/:id/databases/:dbId/backups", "POST .../restore", "GET /backups"] },
              { label: "Secrets", endpoints: ["POST /apps/:id/secrets", "GET /apps/:id/secrets/:key/value", "DELETE /apps/:id/secrets/:key"] },
            ].map((section) => (
              <div key={section.label} className="rounded-md bg-surface-200 p-3">
                <h3 className="mb-1.5 font-medium text-neutral-300">{section.label}</h3>
                <ul className="space-y-0.5">
                  {section.endpoints.map((ep) => (
                    <li key={ep} className="font-mono text-neutral-500">{ep}</li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        {/* Full example */}
        <div>
          <h2 className="mb-3 text-sm font-medium text-white">Complete Example: Next.js + PostgreSQL + Auth</h2>
          <CodeBlock
            lang="typescript"
            code={`// app/api/todos/route.ts — Example API route
import { Pool } from 'pg';

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,  // Auto-injected by Zenith
});

export async function GET(req: Request) {
  // Verify Zenith Auth token
  const token = req.headers.get('Authorization')?.split(' ')[1];
  if (!token) return Response.json({ error: 'Unauthorized' }, { status: 401 });

  const { rows } = await pool.query('SELECT * FROM todos ORDER BY created_at DESC');
  return Response.json(rows);
}

export async function POST(req: Request) {
  const { title } = await req.json();
  const { rows } = await pool.query(
    'INSERT INTO todos (title) VALUES ($1) RETURNING *',
    [title]
  );
  return Response.json(rows[0], { status: 201 });
}`}
          />
        </div>
      </div>
    </Shell>
  );
}
