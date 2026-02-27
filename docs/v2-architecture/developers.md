# Zenith V2 Developer Environment Setup

This document lists all the command-line tools you need to install on your local machine (Mac/Linux) to interact with the Zenith V2 infrastructure. 

All tools can be installed via **Homebrew**.

## 1. Infrastructure Management

Tools required to provision and configure servers.

```bash
# Terraform: Used to provision Hetzner servers and deploy Kubernetes Helm charts
# (Note: HashiCorp changed their license, so we use their official tap instead of core brew)
brew tap hashicorp/tap
brew install hashicorp/tap/terraform

# Ansible: Used to SSH into Hetzner servers and install k3s/Cilium binaries
brew install ansible
```

## 2. Kubernetes Core

Tools required to interact with the k3s cluster.

```bash
# kubectl: The official Kubernetes command-line tool
brew install kubectl

# Helm: The package manager for Kubernetes (used by Terraform, but good for manual debugging)
brew install helm
```

## 3. Security & GitOps (V2)

Tools required for encrypting passwords and secrets before pushing to Git.

```bash
# kubeseal: Used to encrypt plain-text Secrets into SealedSecrets for safe GitOps commits
brew install kubeseal
```

## 4. Networking & Observability (V2)

Tools required to interact with the Cilium network and view live traffic flows.

```bash
# cilium-cli: Used to check the health and status of the Cilium CNI routing
brew install cilium-cli

# hubble: The observability tool used to monitor live network flows and dropped packets
brew install hubble
```

## 5. Continuous Delivery (V2)

Tool for interacting with the GitOps deployment engine from your terminal.

```bash
# argocd: Used to manually sync apps, check app health, or manage the ArgoCD server remotely
brew install argocd
```

---

### Verification
After running the brew commands above, you can verify your environment is ready by checking the version numbers of all the tools:

```bash
terraform --version
ansible --version
kubectl version --client
helm version
kubeseal --version
cilium version
hubble version
argocd version --client
```

---

## 6. How to Use Kubeseal (GitOps Secure Secrets)

**Why:** ArgoCD deploys all apps directly from GitHub. You should **never** commit raw Kubernetes `Secret` YAMLs to GitHub, as they are only base64 encoded and can be stolen easily. 
**What:** `kubeseal` mathematically encrypts your raw Kubernetes `Secret` into a `SealedSecret` using a public key downloaded from the cluster. A `SealedSecret` is safe to commit to GitHub. Only the `sealed-secrets-controller` inside that specific cluster holds the private key to decrypt it back into a usable `Secret`.

### Workflow Example

Imagine you need to add a Stripe API Key to the `zenith-api` application in the `staging` environment.

1. **Ensure you are connected to the correct cluster:**
   `kubeseal` automatically connects to whatever cluster your `kubectl` is currently pointed at (via your `~/.kube/config`).
   ```bash
   kubectl config use-context zenith-staging
   ```

2. **Create the raw secret locally (DO NOT COMMIT THIS FILE):**
   Create a file named `secret.yaml`:
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: payment-secret
     namespace: zenith-staging
   stringData:
     stripe-key: "sk_live_1234567890"
   ```

3. **Seal the secret:**
   Run `kubeseal`. It will automatically connect to the `sealed-secrets-controller` in the cluster, download the public key, and encrypt the file.
   ```bash
   # Read secret.yaml, mathematically encrypt it, write to sealed-secret.yaml
   kubeseal -f secret.yaml -w sealed-secret.yaml \
     --controller-name sealed-secrets \
     --controller-namespace sealed-secrets
   ```

4. **Clean up:**
   The `secret.yaml` file on your laptop contains plain-text passwords. Delete it permanently.
   ```bash
   rm secret.yaml
   ```

5. **Commit to Git:**
   The new `sealed-secret.yaml` file looks like a random string of characters (e.g., `AgCsd4rT5fG...`). Commit this file to GitHub and push. ArgoCD will deploy it, and the cluster will decrypt it safely.

### Handling Multiple Environments (Stage vs Prod)
The encryption keys in the **Staging** cluster are mathematically different from the keys in the **Production** cluster.
A `SealedSecret` encrypted for Staging **cannot** be decrypted by Production.

When sealing secrets for Production, simply change your `kubectl` context to Production before running `kubeseal`:
```bash
# Point to Production cluster
kubectl config use-context zenith-production

# Kubeseal will automatically download the Production public key instead!
kubeseal -f secret-prod.yaml -w sealed-secret-prod.yaml \
  --controller-name sealed-secrets \
  --controller-namespace sealed-secrets
```
