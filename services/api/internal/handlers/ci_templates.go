package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// CITemplateHandler serves CI/CD template files.
type CITemplateHandler struct{}

// NewCITemplateHandler creates a new CITemplateHandler.
func NewCITemplateHandler() *CITemplateHandler {
	return &CITemplateHandler{}
}

// GetTemplate handles GET /ci-templates/:framework
func (h *CITemplateHandler) GetTemplate(c *fiber.Ctx) error {
	framework := c.Params("framework")
	project := c.Query("project", "<your-project>")
	service := c.Query("service", "<your-service>")

	tmpl, ok := ciTemplates[framework]
	if !ok {
		return NewBadRequest("unsupported framework: " + framework + ". Supported: go, nextjs, python, nodejs, rust")
	}

	// Replace placeholders
	result := strings.ReplaceAll(tmpl, "<your-project>", project)
	result = strings.ReplaceAll(result, "<your-service>", service)

	c.Set("Content-Type", "text/yaml; charset=utf-8")
	return c.SendString(result)
}

// ListTemplates handles GET /ci-templates
func (h *CITemplateHandler) ListTemplates(c *fiber.Ctx) error {
	frameworks := make([]string, 0, len(ciTemplates))
	for k := range ciTemplates {
		frameworks = append(frameworks, k)
	}
	return c.JSON(fiber.Map{
		"frameworks": frameworks,
	})
}

var ciTemplates = map[string]string{
	"go": `name: Deploy to Zenith
on:
  push:
    branches: [main]
env:
  REGISTRY: registry.stage.freezenith.com
  PROJECT: <your-project>
  SERVICE: <your-service>
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Build
        run: go build -o app ./...
      - name: Login to Registry
        run: echo "${{ secrets.ZENITH_REGISTRY_PASS }}" | docker login $REGISTRY -u "${{ secrets.ZENITH_REGISTRY_USER }}" --password-stdin
      - name: Build & Push Image
        run: |
          docker build -t $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} .
          docker push $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }}
          docker tag $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} $REGISTRY/$PROJECT/$SERVICE:latest
          docker push $REGISTRY/$PROJECT/$SERVICE:latest
`,

	"nextjs": `name: Deploy to Zenith
on:
  push:
    branches: [main]
env:
  REGISTRY: registry.stage.freezenith.com
  PROJECT: <your-project>
  SERVICE: <your-service>
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Install Dependencies
        run: npm ci
      - name: Build
        run: npm run build
      - name: Login to Registry
        run: echo "${{ secrets.ZENITH_REGISTRY_PASS }}" | docker login $REGISTRY -u "${{ secrets.ZENITH_REGISTRY_USER }}" --password-stdin
      - name: Build & Push Image
        run: |
          docker build -t $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} .
          docker push $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }}
          docker tag $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} $REGISTRY/$PROJECT/$SERVICE:latest
          docker push $REGISTRY/$PROJECT/$SERVICE:latest
`,

	"python": `name: Deploy to Zenith
on:
  push:
    branches: [main]
env:
  REGISTRY: registry.stage.freezenith.com
  PROJECT: <your-project>
  SERVICE: <your-service>
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - name: Install Dependencies
        run: pip install -r requirements.txt
      - name: Login to Registry
        run: echo "${{ secrets.ZENITH_REGISTRY_PASS }}" | docker login $REGISTRY -u "${{ secrets.ZENITH_REGISTRY_USER }}" --password-stdin
      - name: Build & Push Image
        run: |
          docker build -t $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} .
          docker push $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }}
          docker tag $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} $REGISTRY/$PROJECT/$SERVICE:latest
          docker push $REGISTRY/$PROJECT/$SERVICE:latest
`,

	"nodejs": `name: Deploy to Zenith
on:
  push:
    branches: [main]
env:
  REGISTRY: registry.stage.freezenith.com
  PROJECT: <your-project>
  SERVICE: <your-service>
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Install Dependencies
        run: npm ci
      - name: Login to Registry
        run: echo "${{ secrets.ZENITH_REGISTRY_PASS }}" | docker login $REGISTRY -u "${{ secrets.ZENITH_REGISTRY_USER }}" --password-stdin
      - name: Build & Push Image
        run: |
          docker build -t $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} .
          docker push $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }}
          docker tag $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} $REGISTRY/$PROJECT/$SERVICE:latest
          docker push $REGISTRY/$PROJECT/$SERVICE:latest
`,

	"rust": `name: Deploy to Zenith
on:
  push:
    branches: [main]
env:
  REGISTRY: registry.stage.freezenith.com
  PROJECT: <your-project>
  SERVICE: <your-service>
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Rust
        uses: dtolnay/rust-toolchain@stable
      - name: Build
        run: cargo build --release
      - name: Login to Registry
        run: echo "${{ secrets.ZENITH_REGISTRY_PASS }}" | docker login $REGISTRY -u "${{ secrets.ZENITH_REGISTRY_USER }}" --password-stdin
      - name: Build & Push Image
        run: |
          docker build -t $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} .
          docker push $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }}
          docker tag $REGISTRY/$PROJECT/$SERVICE:${{ github.sha }} $REGISTRY/$PROJECT/$SERVICE:latest
          docker push $REGISTRY/$PROJECT/$SERVICE:latest
`,
}
