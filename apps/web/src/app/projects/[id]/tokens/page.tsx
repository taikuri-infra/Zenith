"use client";

import { Shell } from "@/components/shell";
import { useToast } from "@/components/toast";
import { getApi } from "@/lib/get-api";
import type { DeployToken } from "@/lib/api";
import { DEPLOY_TOKEN_SCOPES, DEPLOY_TOKEN_EXPIRY_OPTIONS } from "@/lib/api";
import { useParams } from "next/navigation";
import { useState, useEffect, useCallback } from "react";
import {
  KeyRound,
  Plus,
  Copy,
  Check,
  Trash2,
  RotateCw,
  Clock,
  Shield,
  AlertTriangle,
} from "lucide-react";

const SCOPES = DEPLOY_TOKEN_SCOPES;
const EXPIRY_OPTIONS = DEPLOY_TOKEN_EXPIRY_OPTIONS;

export default function DeployTokensPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { toast } = useToast();
  const api = getApi();

  const [tokens, setTokens] = useState<DeployToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [selectedScopes, setSelectedScopes] = useState<string[]>([
    "deploy:staging",
  ]);
  const [expiresIn, setExpiresIn] = useState("90d");
  const [creating, setCreating] = useState(false);
  const [newToken, setNewToken] = useState<DeployToken | null>(null);
  const [copied, setCopied] = useState<string | null>(null);

  const fetchTokens = useCallback(async () => {
    try {
      const resp = await api.deployTokens.list(projectId);
      setTokens(resp.tokens || []);
    } catch {
      toast("error", "Failed to load deploy tokens");
    } finally {
      setLoading(false);
    }
  }, [projectId, api, toast]);

  useEffect(() => {
    fetchTokens();
  }, [fetchTokens]);

  const handleCreate = async () => {
    if (!name.trim()) {
      toast("error", "Name is required");
      return;
    }
    if (selectedScopes.length === 0) {
      toast("error", "Select at least one scope");
      return;
    }

    setCreating(true);
    try {
      const token = await api.deployTokens.create(
        projectId,
        name.trim(),
        selectedScopes,
        expiresIn
      );
      setNewToken(token);
      setName("");
      setSelectedScopes(["deploy:staging"]);
      fetchTokens();
    } catch (e) {
      toast("error", `Failed to create token: ${e}`);
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (tokenId: string) => {
    if (!confirm("Are you sure you want to revoke this token?")) return;
    try {
      await api.deployTokens.revoke(projectId, tokenId);
      toast("success", "Token revoked");
      fetchTokens();
    } catch {
      toast("error", "Failed to revoke token");
    }
  };

  const handleRotate = async (tokenId: string) => {
    if (
      !confirm(
        "Rotate this token? The old secret will remain valid for 24 hours."
      )
    )
      return;
    try {
      const rotated = await api.deployTokens.rotate(projectId, tokenId);
      setNewToken(rotated);
      toast("success", "Token rotated — old secret valid for 24h");
      fetchTokens();
    } catch {
      toast("error", "Failed to rotate token");
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(null), 2000);
  };

  return (
    <Shell>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-white">Deploy Tokens</h1>
            <p className="text-sm text-neutral-500">
              Create tokens for CI/CD pipelines (GitHub Actions, GitLab CI)
            </p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white hover:bg-accent-500"
          >
            <Plus className="h-4 w-4" />
            New Token
          </button>
        </div>

        {/* New token created — show secret once */}
        {newToken?.secret && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 space-y-3">
            <div className="flex items-center gap-2 text-amber-400 text-sm font-medium">
              <AlertTriangle className="h-4 w-4" />
              Save these credentials — the secret won&apos;t be shown again
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <span className="text-xs text-neutral-500 w-24">Token ID:</span>
                <code className="flex-1 text-xs bg-neutral-900 rounded px-2 py-1 text-neutral-300">
                  {newToken.token_id}
                </code>
                <button
                  onClick={() =>
                    copyToClipboard(newToken.token_id, "token_id")
                  }
                  className="text-neutral-500 hover:text-white"
                >
                  {copied === "token_id" ? (
                    <Check className="h-4 w-4 text-green-400" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </button>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-neutral-500 w-24">Secret:</span>
                <code className="flex-1 text-xs bg-neutral-900 rounded px-2 py-1 text-neutral-300 break-all">
                  {newToken.secret}
                </code>
                <button
                  onClick={() =>
                    copyToClipboard(newToken.secret!, "secret")
                  }
                  className="text-neutral-500 hover:text-white"
                >
                  {copied === "secret" ? (
                    <Check className="h-4 w-4 text-green-400" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </button>
              </div>
            </div>
            <div className="text-xs text-neutral-500 mt-2">
              Add these to your GitHub repo: Settings &rarr; Secrets &rarr;{" "}
              <code>ZENITH_TOKEN_ID</code> and <code>ZENITH_TOKEN_SECRET</code>
            </div>
            <button
              onClick={() => setNewToken(null)}
              className="text-xs text-neutral-500 hover:text-white"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Create form */}
        {showCreate && (
          <div className="rounded-lg border border-border bg-card p-4 space-y-4">
            <h3 className="text-sm font-medium text-white">
              Create Deploy Token
            </h3>
            <input
              type="text"
              placeholder="Token name (e.g., GitHub Actions)"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded-lg border border-border bg-neutral-900 px-3 py-2 text-sm text-white"
            />

            <div>
              <label className="text-xs text-neutral-500 block mb-2">
                Scopes
              </label>
              <div className="flex flex-wrap gap-2">
                {SCOPES.map((scope) => (
                  <button
                    key={scope.value}
                    onClick={() =>
                      setSelectedScopes((prev) =>
                        prev.includes(scope.value)
                          ? prev.filter((s) => s !== scope.value)
                          : [...prev, scope.value]
                      )
                    }
                    className={`rounded-full px-3 py-1 text-xs border ${
                      selectedScopes.includes(scope.value)
                        ? "border-accent-500 bg-accent-500/10 text-accent-400"
                        : "border-border text-neutral-500 hover:text-white"
                    }`}
                  >
                    {scope.label}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="text-xs text-neutral-500 block mb-2">
                Expires in
              </label>
              <select
                value={expiresIn}
                onChange={(e) => setExpiresIn(e.target.value)}
                className="rounded-lg border border-border bg-neutral-900 px-3 py-2 text-sm text-white"
              >
                {EXPIRY_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex gap-2">
              <button
                onClick={handleCreate}
                disabled={creating}
                className="rounded-lg bg-accent-600 px-4 py-2 text-sm font-medium text-white hover:bg-accent-500 disabled:opacity-50"
              >
                {creating ? "Creating..." : "Create Token"}
              </button>
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm text-neutral-400 hover:text-white"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Token list */}
        {loading ? (
          <div className="text-sm text-neutral-500">Loading...</div>
        ) : tokens.length === 0 ? (
          <div className="text-center py-12 text-neutral-500">
            <KeyRound className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No deploy tokens yet</p>
            <p className="text-xs mt-1">
              Create a token to deploy from GitHub Actions or other CI systems
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {tokens.map((token) => (
              <div
                key={token.id}
                className="rounded-lg border border-border bg-card p-4"
              >
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <KeyRound className="h-4 w-4 text-accent-400" />
                      <span className="text-sm font-medium text-white">
                        {token.name}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 text-xs text-neutral-500">
                      <span>
                        ID: <code>{token.token_id}</code>
                      </span>
                      <span>
                        Secret: <code>{token.token_prefix}...</code>
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      {token.scopes?.map((scope) => (
                        <span
                          key={scope}
                          className="rounded-full bg-neutral-800 px-2 py-0.5 text-xs text-neutral-400"
                        >
                          {scope}
                        </span>
                      ))}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="text-right text-xs text-neutral-500 mr-2">
                      {token.last_used_at ? (
                        <div className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          Last used{" "}
                          {new Date(token.last_used_at).toLocaleDateString()}
                        </div>
                      ) : (
                        <span>Never used</span>
                      )}
                      {token.expires_at && (
                        <div className="flex items-center gap-1 mt-0.5">
                          <Shield className="h-3 w-3" />
                          Expires{" "}
                          {new Date(token.expires_at).toLocaleDateString()}
                        </div>
                      )}
                    </div>
                    <button
                      onClick={() => handleRotate(token.id)}
                      title="Rotate secret"
                      className="rounded p-1.5 text-neutral-500 hover:bg-neutral-800 hover:text-white"
                    >
                      <RotateCw className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleRevoke(token.id)}
                      title="Revoke token"
                      className="rounded p-1.5 text-neutral-500 hover:bg-red-500/10 hover:text-red-400"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Usage guide */}
        <div className="rounded-lg border border-border bg-card p-4 space-y-3">
          <h3 className="text-sm font-medium text-white">
            GitHub Actions Usage
          </h3>
          <pre className="text-xs bg-neutral-900 rounded-lg p-3 text-neutral-400 overflow-x-auto">
            {`# .github/workflows/deploy.yml
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: docker build -t myrepo/app:\${{ github.sha }} .
      - name: Push
        run: docker push myrepo/app:\${{ github.sha }}
      - name: Deploy to Staging
        uses: dotechhq/zenith-stage@v1
        with:
          token-id: \${{ secrets.ZENITH_TOKEN_ID }}
          token-secret: \${{ secrets.ZENITH_TOKEN_SECRET }}
          app: my-app
          image: myrepo/app:\${{ github.sha }}`}
          </pre>
        </div>
      </div>
    </Shell>
  );
}
