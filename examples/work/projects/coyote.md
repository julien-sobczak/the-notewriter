# Coyote

## Overview: Presentation

Coyote is the main API service of the Acme Platform, serving as the gateway for all client applications. Built with Go for high performance and reliability, Coyote provides a comprehensive RESTful API that handles authentication, authorization, data management, and business logic.

Key features:
- RESTful API with OpenAPI specification
- JWT-based authentication
- Role-based access control (RBAC)
- Request rate limiting and throttling
- Comprehensive audit logging
- Multi-tenant support
- Real-time WebSocket connections
- Integration with third-party services

The API is designed to support web, mobile, and desktop clients with consistent behavior across platforms. It maintains high availability through horizontal scaling and implements circuit breakers for resilient communication with downstream services.

## Tasks

### Todo: Implement GraphQL endpoint 📅 ❗️

`@status: planned` `@due: 2026-02-15` `@priority: high`

Add GraphQL support alongside REST API to provide more flexible data querying for complex client requirements.

### Todo: Add API versioning 🔼

`@status: todo` `@priority: medium`

Implement proper API versioning strategy to support backward compatibility and smooth migrations.

### Todo: Optimize authentication middleware ⏱️ 🔼

`@status: in-progress` `@priority: medium`

Reduce authentication overhead by implementing token caching and optimizing database queries.

### Todo: Implement request tracing 📝 🔼

`@status: todo` `@priority: medium`

Add distributed tracing support using OpenTelemetry for better observability across microservices.

### Todo: Write API documentation 📅 🔼

`@status: in-progress` `@due: 2026-01-25` `@priority: medium`

Complete OpenAPI specification and generate interactive API documentation using Swagger UI.

## Checklist: Deployment

1. Review and merge approved pull requests
2. Update CHANGELOG.md with new features and fixes
3. Bump version number in version.go
4. Run linter: `make lint`
5. Run unit tests: `make test`
6. Run integration tests: `make test-integration`
7. Build production binary: `make build-prod`
8. Create Docker image: `docker build -t coyote:latest -f Dockerfile.prod .`
9. Run security scan: `trivy image coyote:latest`
10. Tag image: `docker tag coyote:latest coyote:v2.3.4`
11. Push to registry: `docker push coyote:v2.3.4`
12. Update Helm chart values with new version
13. Deploy to staging: `helm upgrade coyote ./charts/coyote -f values-staging.yaml`
14. Run smoke tests against staging
15. Deploy to production: `helm upgrade coyote ./charts/coyote -f values-production.yaml --wait`
16. Monitor error rates and latency in Datadog
17. Verify health checks: `curl https://api.acme.corp/health`
18. Announce deployment in #deployments Slack channel

## Cheatsheet: Check API Health

To verify API health and status:

```bash
# Basic health check
curl https://api.acme.corp/health

# Detailed health status with dependencies
curl https://api.acme.corp/health/detailed

# Check API version
curl https://api.acme.corp/version

# Test authentication endpoint
curl -X POST https://api.acme.corp/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@acme.corp","password":"test123"}'
```

## Cheatsheet: View Active Connections

To monitor active connections and performance:

```bash
# Connect to Coyote pod
kubectl exec -it coyote-xyz -- sh

# View active connections (requires metrics endpoint)
curl localhost:9090/metrics | grep http_requests_total

# Check connection pool status
curl localhost:9090/debug/pool

# View recent access logs
kubectl logs -f deployment/coyote --tail=100

# Filter for specific endpoint
kubectl logs deployment/coyote | grep "/api/v1/users"
```

## Cheatsheet: Enable Debug Logging

To enable detailed debug logging for troubleshooting:

```bash
# Edit configmap
kubectl edit configmap coyote-config

# Change log_level from "info" to "debug"
# Save and exit

# Restart pods to pick up new config
kubectl rollout restart deployment/coyote

# Watch debug logs
kubectl logs -f deployment/coyote | grep DEBUG

# Remember to set back to "info" after debugging
```

## Cheatsheet: Test Rate Limiting

To verify rate limiting is working correctly:

```bash
# Test with multiple rapid requests
for i in {1..100}; do
  curl -w "\n%{http_code}\n" \
    -H "Authorization: Bearer $TOKEN" \
    https://api.acme.corp/api/v1/users
  sleep 0.1
done

# Should see 429 Too Many Requests after threshold
# Check rate limit headers:
curl -I \
  -H "Authorization: Bearer $TOKEN" \
  https://api.acme.corp/api/v1/users

# Headers should include:
# X-RateLimit-Limit: 100
# X-RateLimit-Remaining: 95
# X-RateLimit-Reset: 1640000000
```

## Cheatsheet: Invalidate Cache

To clear API cache for specific resources:

```bash
# Connect to Redis cache
kubectl exec -it redis-master-0 -- redis-cli

# View cache keys
KEYS coyote:cache:*

# Invalidate specific user cache
DEL coyote:cache:user:12345

# Invalidate all user caches
EVAL "return redis.call('del', unpack(redis.call('keys', 'coyote:cache:user:*')))" 0

# Invalidate entire cache (use with caution!)
FLUSHDB

# Exit Redis CLI
EXIT
```

## Cheatsheet: Database Migration

To apply database migrations:

```bash
# Check current migration status
kubectl exec -it coyote-xyz -- /app/coyote migrate status

# Apply pending migrations (dry-run first)
kubectl exec -it coyote-xyz -- /app/coyote migrate up --dry-run

# Apply migrations for real
kubectl exec -it coyote-xyz -- /app/coyote migrate up

# Rollback last migration if needed
kubectl exec -it coyote-xyz -- /app/coyote migrate down 1

# View migration history
kubectl exec -it coyote-xyz -- /app/coyote migrate history
```
