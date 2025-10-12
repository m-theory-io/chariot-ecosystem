# chariot-ecosystem
Chariot monorepo for go-chariot, charioteer and visual-dsl

## Canonical build and push (Azure)

Use this exact flow for all builds and pushes. Tags are typically versioned (e.g., v0.026).

```
./scripts/build-azure-cross-platform.sh <tag> [all|go-chariot|charioteer|visual-dsl|nginx]
./scripts/push-images.sh <tag> [all|go-chariot|charioteer|visual-dsl|nginx]
```

Notes
- docker-compose.azure.yml defaults: nginx uses tag `amd64` as the default alias; the push script updates that alias when you push a versioned tag.
- The older `deploy-azure.sh` script has been removed to avoid confusion.
