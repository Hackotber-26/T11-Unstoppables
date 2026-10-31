Write-Host "Creating TrainSync Repository Structure..." -ForegroundColor Green

$folders = @(

# =====================================================
# DOCS
# =====================================================

"docs",
"docs/vision",
"docs/architecture",
"docs/algorithms",
"docs/simulation",
"docs/api",
"docs/adr",

# =====================================================
# FRONTEND
# =====================================================

"frontend",
"frontend/public",

"frontend/assets",
"frontend/assets/fonts",
"frontend/assets/icons",
"frontend/assets/svg",
"frontend/assets/logos",
"frontend/assets/animations",
"frontend/assets/illustrations",

# Public Pages

"frontend/pages/public",
"frontend/pages/public/landing",
"frontend/pages/public/about",
"frontend/pages/public/faq",
"frontend/pages/public/pricing",
"frontend/pages/public/contact",
"frontend/pages/public/safety",

# Auth

"frontend/pages/auth",
"frontend/pages/auth/welcome",
"frontend/pages/auth/login",
"frontend/pages/auth/register",
"frontend/pages/auth/otp-verification",
"frontend/pages/auth/consent",
"frontend/pages/auth/onboarding",

# Rider

"frontend/pages/rider",
"frontend/pages/rider/dashboard",
"frontend/pages/rider/pool-request",
"frontend/pages/rider/destination-select",
"frontend/pages/rider/fare-preview",
"frontend/pages/rider/matching",
"frontend/pages/rider/pool-formed",
"frontend/pages/rider/bay-assigned",
"frontend/pages/rider/bay-navigation",
"frontend/pages/rider/live-trip",
"frontend/pages/rider/trip-complete",
"frontend/pages/rider/payment",
"frontend/pages/rider/payment-success",
"frontend/pages/rider/invoice",
"frontend/pages/rider/rating",
"frontend/pages/rider/wallet",
"frontend/pages/rider/subscriptions",
"frontend/pages/rider/trip-history",
"frontend/pages/rider/notifications",
"frontend/pages/rider/support",
"frontend/pages/rider/profile",

# Driver

"frontend/pages/driver",
"frontend/pages/driver/dashboard",
"frontend/pages/driver/availability",
"frontend/pages/driver/assignment",
"frontend/pages/driver/bay-arrival",
"frontend/pages/driver/otp-check",
"frontend/pages/driver/navigation",
"frontend/pages/driver/trip-summary",
"frontend/pages/driver/earnings",
"frontend/pages/driver/verification",
"frontend/pages/driver/ratings",
"frontend/pages/driver/profile",

# Admin

"frontend/pages/admin",
"frontend/pages/admin/dashboard",
"frontend/pages/admin/users",
"frontend/pages/admin/drivers",
"frontend/pages/admin/stations",
"frontend/pages/admin/bays",
"frontend/pages/admin/vehicles",
"frontend/pages/admin/pools",
"frontend/pages/admin/trips",
"frontend/pages/admin/subscriptions",
"frontend/pages/admin/incidents",
"frontend/pages/admin/analytics",
"frontend/pages/admin/reports",

# Simulation

"frontend/pages/simulation",
"frontend/pages/simulation/sandbox",
"frontend/pages/simulation/analytics",
"frontend/pages/simulation/validation",
"frontend/pages/simulation/experiments",

# Styles

"frontend/styles",
"frontend/styles/components",
"frontend/styles/pages",

# Scripts

"frontend/scripts",
"frontend/scripts/core",
"frontend/scripts/auth",
"frontend/scripts/engine",
"frontend/scripts/simulation",
"frontend/scripts/payments",
"frontend/scripts/maps",
"frontend/scripts/rendering",
"frontend/scripts/scroll",
"frontend/scripts/workers",
"frontend/scripts/ui",

# Tests

"frontend/tests",
"frontend/tests/unit",
"frontend/tests/visual",
"frontend/tests/e2e",
"frontend/tests/accessibility",

# =====================================================
# BACKEND
# =====================================================

"backend",

"backend/cmd",
"backend/cmd/api",
"backend/cmd/pooler",
"backend/cmd/worker",
"backend/cmd/simulator",
"backend/cmd/migrate",
"backend/cmd/seed",

"backend/internal",

"backend/internal/auth",
"backend/internal/users",
"backend/internal/profiles",
"backend/internal/riders",
"backend/internal/drivers",
"backend/internal/stations",
"backend/internal/metro",
"backend/internal/bays",
"backend/internal/vehicles",
"backend/internal/matching",
"backend/internal/pooling",
"backend/internal/routing",
"backend/internal/fares",
"backend/internal/safher",
"backend/internal/payments",
"backend/internal/wallet",
"backend/internal/subscriptions",
"backend/internal/trips",
"backend/internal/incidents",
"backend/internal/notifications",
"backend/internal/realtime",
"backend/internal/analytics",
"backend/internal/simulation",
"backend/internal/audit",
"backend/internal/api",
"backend/internal/storage",

"backend/migrations",
"backend/queries",

"backend/tests",
"backend/tests/integration",
"backend/tests/load",
"backend/tests/vectors",
"backend/tests/property",

# =====================================================
# DATASETS
# =====================================================

"datasets",
"datasets/synthetic",
"datasets/verified",
"datasets/stations",
"datasets/routes",
"datasets/scenarios",

# =====================================================
# SPECS
# =====================================================

"spec",
"spec/vectors",
"spec/scenarios",
"spec/configs",
"spec/tariffs",

# =====================================================
# TOOLS
# =====================================================

"tools",
"tools/scenario-builder",
"tools/vector-generator",
"tools/benchmark",
"tools/migration-tools",

# =====================================================
# RESEARCH
# =====================================================

"research",
"research/notebooks",
"research/exports",
"research/experiments",

# =====================================================
# INFRASTRUCTURE
# =====================================================

"infrastructure",
"infrastructure/docker",
"infrastructure/caddy",
"infrastructure/postgres",
"infrastructure/redis",
"infrastructure/grafana",
"infrastructure/prometheus",
"infrastructure/monitoring",

# =====================================================
# DEPLOY
# =====================================================

"deploy",
"deploy/dev",
"deploy/staging",
"deploy/production",

# =====================================================
# GITHUB
# =====================================================

".github",
".github/workflows"
)

foreach ($folder in $folders) {
    New-Item -ItemType Directory -Force -Path $folder | Out-Null
}

# =====================================================
# ROOT FILES
# =====================================================

$files = @(
"README.md",
"LICENSE",
"CONTRIBUTING.md",
"SECURITY.md",
"CODE_OF_CONDUCT.md",
"CHANGELOG.md",
"ROADMAP.md",
"DISCLAIMER.md",
".gitignore",
".editorconfig",
".gitattributes",

"backend/go.mod",
"backend/go.sum",

"frontend/index.html",

"docs/vision/executive-summary.md",
"docs/vision/product-vision.md",
"docs/vision/problem-statement.md",
"docs/vision/business-model.md",
"docs/vision/roadmap.md",

"docs/architecture/system-overview.md",
"docs/architecture/frontend-architecture.md",
"docs/architecture/backend-architecture.md",
"docs/architecture/simulation-architecture.md",
"docs/architecture/data-architecture.md",
"docs/architecture/payment-architecture.md",
"docs/architecture/safety-architecture.md",
"docs/architecture/deployment-architecture.md",

"docs/api/openapi.yml",

".github/workflows/lint.yml",
".github/workflows/unit-tests.yml",
".github/workflows/integration-tests.yml",
".github/workflows/build.yml",
".github/workflows/deploy.yml",
".github/workflows/release.yml"
)

foreach ($file in $files) {

    $directory = Split-Path $file -Parent

    if ($directory -ne "") {
        New-Item -ItemType Directory -Path $directory -Force | Out-Null
    }

    if (!(Test-Path $file)) {
        New-Item -ItemType File -Path $file -Force | Out-Null
    }
}

# =====================================================
# GITKEEP
# =====================================================

Get-ChildItem -Directory -Recurse | ForEach-Object {

    $gitkeep = Join-Path $_.FullName ".gitkeep"

    if (!(Test-Path $gitkeep)) {
        New-Item -ItemType File -Path $gitkeep -Force | Out-Null
    }
}

Write-Host ""
Write-Host "======================================" -ForegroundColor Green
Write-Host "TrainSync Repository Created" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green
Write-Host ""

tree /Fck