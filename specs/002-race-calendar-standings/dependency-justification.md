# Dependency Justification Ledger

All third-party dependencies must be listed with owner, purpose, and justification per Constitution §5 (Delivery & Dependency Discipline).

## Backend (Go)

| Dependency | Version | Owner | Purpose | Justification |
|---|---|---|---|---|
| `github.com/go-chi/chi/v5` | v5.2.5 | @karlkuhnhausen | HTTP router | Lightweight, stdlib-compatible, idiomatic Go router. Avoids heavier frameworks. |
| `github.com/Azure/azure-sdk-for-go/sdk/azcore` | v1.21.1 | @karlkuhnhausen | Azure SDK core | Required by all Azure service clients. Official Microsoft SDK. |
| `github.com/Azure/azure-sdk-for-go/sdk/azidentity` | v1.13.1 | @karlkuhnhausen | Azure credential | DefaultAzureCredential for Managed Identity + local dev. Official Microsoft SDK. |
| `github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos` | v1.4.2 | @karlkuhnhausen | Cosmos DB client | Direct database access. Official Microsoft SDK. |
| `github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets` | v1.4.0 | @karlkuhnhausen | Key Vault secrets | Secret loading at startup. Official Microsoft SDK. |

## Frontend (React/TypeScript)

| Dependency | Version | Owner | Purpose | Justification |
|---|---|---|---|---|
| `react` | ^18.3.1 | @karlkuhnhausen | UI framework | Constitution-mandated stack. |
| `react-dom` | ^18.3.1 | @karlkuhnhausen | DOM rendering | Required by React. |
| `vite` | ^5.4.8 | @karlkuhnhausen | Build tool | Fast dev server and production bundler. Constitution-mandated. |
| `typescript` | ^5.6.2 | @karlkuhnhausen | Type system | Constitution-mandated stack. |

## Policy

- No dependency may be added without an entry in this ledger.
- Transitive dependencies are covered by their parent entry.
- Review quarterly or when upgrading major versions.
