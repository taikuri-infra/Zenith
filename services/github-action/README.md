# zenith-deploy-action

Deploy your pre-built Docker image to [Zenith Cloud](https://freezenith.com) from GitHub Actions.

## Usage

```yaml
- name: Deploy to Zenith
  uses: taikuri-infra/zenith-deploy-action@v1
  with:
    token-id: ${{ secrets.ZENITH_TOKEN_ID }}
    token-secret: ${{ secrets.ZENITH_TOKEN_SECRET }}
    app: my-app
    image: ghcr.io/my-org/my-app:${{ github.sha }}
    environment: production  # or: staging
```

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `token-id` | ✅ | — | Deploy token ID (`znt_id_...`) from Zenith dashboard |
| `token-secret` | ✅ | — | Deploy token secret (`znt_sk_...`) from Zenith dashboard |
| `app` | ✅ | — | Application name on Zenith |
| `image` | ✅ | — | Docker image URL with tag |
| `environment` | ❌ | `production` | Target environment: `staging` or `production` |
| `api-url` | ❌ | `https://api.freezenith.com` | Zenith API URL |
| `wait` | ❌ | `true` | Wait for deployment to be healthy |
| `timeout` | ❌ | `180` | Deployment timeout in seconds |

## Outputs

| Output | Description |
|--------|-------------|
| `deployment-id` | The deployment ID |
| `url` | The app URL after deployment |

## Full Example

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build and push image
        run: |
          docker build -t ghcr.io/${{ github.repository }}:${{ github.sha }} .
          docker push ghcr.io/${{ github.repository }}:${{ github.sha }}

      - name: Deploy to staging
        uses: taikuri-infra/zenith-deploy-action@v1
        with:
          token-id: ${{ secrets.ZENITH_TOKEN_ID }}
          token-secret: ${{ secrets.ZENITH_TOKEN_SECRET }}
          app: my-app
          image: ghcr.io/${{ github.repository }}:${{ github.sha }}
          environment: staging

      - name: Deploy to production
        if: github.ref == 'refs/heads/main'
        uses: taikuri-infra/zenith-deploy-action@v1
        with:
          token-id: ${{ secrets.ZENITH_TOKEN_ID }}
          token-secret: ${{ secrets.ZENITH_TOKEN_SECRET }}
          app: my-app
          image: ghcr.io/${{ github.repository }}:${{ github.sha }}
          environment: production
```

## Setup

1. Go to your Zenith dashboard → Project → Deploy Tokens
2. Create a token with `deploy:staging` and/or `deploy:production` scopes
3. Add `ZENITH_TOKEN_ID` and `ZENITH_TOKEN_SECRET` to your GitHub repository secrets
