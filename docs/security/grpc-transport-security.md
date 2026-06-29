# gRPC Transport Security

## Decision: Plaintext transport with application-layer service identity

All internal gRPC connections in POS-Stery use `insecure.NewCredentials()` (plaintext HTTP/2).
This document records why that is correct for this architecture and what conditions would require change.

---

## Current gRPC Architecture

### Servers (all use `_shared/server.New`)

| Service | Port | Caller |
|---------|------|--------|
| auth-service | 8081 | api-gateway |
| product-service | 8082 | api-gateway |
| inventory-service | 8083 | api-gateway, sales-service, supplier-service |
| sales-service | 8084 | api-gateway |
| supplier-service | 8085 | api-gateway |
| customer-service | 8086 | api-gateway |

### Clients

| File | Connects to |
|------|-------------|
| `services/api-gateway/internal/clients/base.go` (`Dial`) | All 6 services above |
| `services/sales-service/internal/infrastructure/grpc_clients/inventory_client.go` | inventory-service |
| `services/supplier-service/internal/infrastructure/grpc_clients/inventory_client.go` | inventory-service |

---

## Trust Boundaries

```
Internet
    │  HTTPS (TLS terminated at ingress/load balancer)
    ▼
[ HTTP API Gateway :8000 ]        ← only public-facing component
    │  plaintext HTTP/2 gRPC
    ▼
[ pos-net / K8s pod network ]     ← trust boundary
    ├─ auth-service :8081
    ├─ product-service :8082
    ├─ inventory-service :8083  ← also called by sales, supplier
    ├─ sales-service :8084
    ├─ supplier-service :8085
    └─ customer-service :8086
```

No gRPC port is reachable from outside the trust boundary. External clients
only interact with the HTTP gateway, which speaks REST/JSON, not gRPC.

---

## Internal vs External Traffic

| Traffic type | Protocol | TLS |
|---|---|---|
| Browser / mobile → gateway | HTTPS | Yes — at ingress/LB |
| Gateway → backend services | gRPC H2 plaintext | No — internal network |
| sales-service → inventory-service | gRPC H2 plaintext | No — internal network |
| supplier-service → inventory-service | gRPC H2 plaintext | No — internal network |

---

## Existing Infrastructure Protections

### Network isolation
- **Development**: Docker bridge network `pos-net`. Containers communicate via Docker DNS (`auth-service:8081`). The bridge is not routable from the host except through explicitly mapped ports.
- **Production (target)**: Kubernetes overlay network (CNI). Pod-to-pod traffic is isolated within the cluster namespace. NetworkPolicy should restrict inter-namespace and ingress access.

### Service identity — X-Service-Token
Every gRPC server validates an `X-Service-Token` HMAC-SHA256 header on every inbound unary call (`_shared/middleware.UnaryInterceptorChain`). The shared secret is injected at runtime via `SERVICE_TOKEN_SECRET`. A pod without the secret cannot successfully call any backend service. This provides **authentication without encryption**.

### Note: development docker-compose host port exposure
`deploy/docker/docker-compose.yml` maps gRPC ports to the Docker host (e.g., `8081:8081`) to allow local `grpcurl` inspection during development. These host mappings **must be removed or restricted to `127.0.0.1`** in any environment accessible from the network. A production-grade compose or Helm chart should omit all gRPC `ports:` entries.

---

## Transport Security Model

| Property | Mechanism |
|---|---|
| Confidentiality | Docker/K8s network isolation (no eavesdropping from outside the network) |
| Service authentication | X-Service-Token HMAC-SHA256 per call |
| JWT authenticity | RS256 — signed with `keys/private.pem`, verified with `keys/public.pem` |
| Wire encryption | **None at the application layer** — relies on network isolation |

This matches the pattern used by most internal microservice deployments that do not yet run a service mesh.

---

## Security Risks and Mitigations

### Risk 1 — Compromised pod on the same network
A container that breaks out of its isolation could eavesdrop on plaintext gRPC traffic.

**Mitigation**: K8s NetworkPolicy restricting pod-to-pod communication to only declared paths. Future: Istio/Linkerd mTLS, which provides per-connection mutual authentication and encryption with automatic certificate rotation.

### Risk 2 — gRPC ports exposed on Docker host in development
Any process on the developer machine can reach the backend gRPC services directly, bypassing the gateway and the X-Service-Token check (if `SERVICE_TOKEN_SECRET` is the dev default `changeme`).

**Mitigation**: Accepted dev-time trade-off. The `changeme` secret must be replaced by a strong random value before any networked deployment. Production docker-compose or Helm charts must remove `ports:` from backend service definitions.

### Risk 3 — Shared HMAC secret for service identity
X-Service-Token uses a single shared secret. A compromised secret allows any holder to impersonate any service.

**Mitigation**: Secret rotation procedure. Future: mTLS with per-service certificates issued by a cluster CA (Istio, cert-manager).

---

## Justification for Current Implementation

1. **Deployment model**: All gRPC servers are internal. The network perimeter (Docker bridge / K8s overlay) is the primary isolation layer — the same trust model used by Kubernetes itself for control-plane components.

2. **HMAC service tokens**: Provide service identity verification without requiring a PKI. This satisfies the threat of an unauthorized caller inside the network.

3. **Operational cost**: Application-level TLS for internal gRPC requires a CA, per-service certificate issuance, rotation, and runtime loading — significantly more complexity than the current threat model warrants. The correct evolution path is a service mesh, not hand-rolled cert loading.

4. **RSA keys in `keys/`**: These are JWT signing keys (RS256), not TLS certificates. Using them for gRPC TLS would be architecturally incorrect; gRPC TLS requires X.509 certificates with appropriate Subject Alternative Names, issued by a separate CA.

5. **Industry precedent**: Most Kubernetes-native architectures run plaintext gRPC internally and rely on a service mesh for wire encryption when required by compliance or threat model.

---

## Future Recommendations

| Priority | Action |
|---|---|
| Before production | Remove gRPC `ports:` host mappings from docker-compose / Helm; restrict to `127.0.0.1` in dev if grpcurl access is needed |
| Before production | Replace `SERVICE_TOKEN_SECRET: changeme` with a randomly generated 256-bit secret injected via Kubernetes Secrets |
| Medium term | Deploy Istio or Linkerd in STRICT mTLS mode — eliminates the need for X-Service-Token and adds per-connection mutual authentication with automatic cert rotation |
| Long term | If gRPC must ever be exposed outside the cluster, replace `insecure.NewCredentials()` with `credentials.NewTLS(tlsCfg)` and provision certificates via cert-manager or a managed CA |

---

## Final Conclusion

**The current implementation is appropriate for this architecture.**

Plaintext gRPC with HMAC-SHA256 service token authentication is correct when:
- All gRPC endpoints are internal to a trusted network boundary ✓
- No gRPC port is exposed externally ✓
- A service identity mechanism (X-Service-Token) is in place ✓
- The production deployment will add network-level controls (K8s NetworkPolicy, eventual service mesh) ✓

The `insecure.NewCredentials()` calls are intentional design decisions, not oversights. Each usage is documented with an inline comment pointing back to this document.

**Trigger for re-evaluation**: If any gRPC port is ever exposed outside the Kubernetes cluster, or if a service mesh is not planned before production launch, application-level TLS must be implemented before that exposure occurs.
