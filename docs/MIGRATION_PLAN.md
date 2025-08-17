# Chariot Ecosystem Migration Plan

## Overview
Migration from separate repos to monorepo structure in `chariot-ecosystem`.

## Files to Migrate

### 1. Core Application Files (from visual-dsl)
**Destination: `services/visual-dsl/`**

#### Essential React/TypeScript files:
- `src/` (entire directory)
- `public/`
- `package.json`
- `package-lock.json`
- `tsconfig.json`
- `vite.config.js`
- `tailwind.config.js`
- `postcss.config.js`
- `eslint.config.js`
- `index.html`
- `README.md`

#### Generated/Modified Components (NEW):
- `src/components/ChariotGeneratorButton.tsx` ✅ (NEW - keep this)

### 2. Docker Infrastructure (Recreate Clean)
**Destination: `infrastructure/docker/`**

#### Docker files to recreate (using lessons learned):
- `docker-compose.yml` (main ecosystem)
- `docker-compose.dev.yml` (development overrides)
- `infrastructure/docker/visual-dsl/Dockerfile`
- `infrastructure/docker/go-chariot/Dockerfile`
- `infrastructure/docker/charioteer/Dockerfile`

#### Database configurations:
- `databases/mysql/init.sql`
- `databases/mysql/my.cnf`
- `databases/couchbase/init.sh`

#### Nginx and SSL:
- `infrastructure/docker/nginx/nginx.conf`
- SSL certificate generation scripts

### 3. Build and Development Scripts
**Destination: `scripts/`**

#### Scripts to recreate:
- `scripts/setup-ecosystem.sh`
- `scripts/dev-start.sh`
- `scripts/build-all.sh`
- `scripts/deploy.sh`
- `Makefile` (unified for all services)

### 4. Configuration Files
**Destination: `config/`**

#### Environment and config:
- `.env.example` (template)
- Development configurations
- Production configurations

## Files to EXCLUDE (artifacts/experiments)

### Docker Experiments:
- `docker/` (current structure - too complex)
- `docker-compose.ecosystem.yml` (overly complex)
- `test-network.yml`
- `test-go-chariot`

### Temporary/Test Files:
- `cmd/` (simple stub - replace with real go-chariot)
- `go.mod`, `go.sum` (belongs in go-chariot service)
- `chariot_demo.html`
- `test_generator.html`
- Various `.md` documentation files (merge into main docs)

### Generated/Temporary:
- `static/`
- `templates/`
- `expected_output.ch`

## Migration Steps

### Phase 1: Service Migration
1. Copy visual-dsl core app to `services/visual-dsl/`
2. Copy go-chariot repo to `services/go-chariot/`
3. Copy charioteer repo to `services/charioteer/` (if separate)

### Phase 2: Infrastructure Setup
1. Create clean docker structure in `infrastructure/docker/`
2. Set up databases configuration in `databases/`
3. Create unified docker-compose files

### Phase 3: Build System
1. Create unified Makefile
2. Set up development scripts
3. Configure CI/CD (if needed)

### Phase 4: Documentation
1. Consolidate all documentation
2. Create development guide
3. Update README files

## Key Improvements in Monorepo

### Simplified Docker Structure:
```yaml
# docker-compose.yml (simplified)
services:
  visual-dsl:
    build: 
      context: ./services/visual-dsl
      dockerfile: ../../infrastructure/docker/visual-dsl/Dockerfile
    
  go-chariot:
    build:
      context: ./services/go-chariot
      dockerfile: ../../infrastructure/docker/go-chariot/Dockerfile
    
  charioteer:
    build:
      context: ./services/charioteer  
      dockerfile: ../../infrastructure/docker/charioteer/Dockerfile
```

### Host Networking Solution:
- Use host networking for all services (lesson learned)
- Eliminate Docker bridge network conflicts
- Use localhost URLs for inter-service communication

### Azure Configuration:
- Centralized Azure Key Vault configuration
- Tenant ID: `82fbfa53-3046-4f39-a182-f5e0082313d4`
- Vault URL: `https://chariot-vault.vault.azure.net`
- Mount Azure CLI credentials properly

## Next Actions

1. ✅ Create this migration plan
2. ⏳ Create migration scripts
3. ⏳ Generate new docker-compose structure
4. ⏳ Create file copy commands
5. ⏳ Test migration in chariot-ecosystem repo
