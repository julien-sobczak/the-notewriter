# Runner

## Overview: Presentation

Runner is the background worker service of the Acme Platform, responsible for processing asynchronous tasks and batch operations. Built with scalability and reliability in mind, Runner consumes messages from the message queue and executes long-running operations without blocking the main API.

Key features:
- Asynchronous task processing
- Job scheduling and cron-like functionality
- Parallel execution with configurable worker pools
- Automatic retry mechanism with exponential backoff
- Comprehensive logging and monitoring
- Graceful shutdown and task recovery

The service is designed to handle various types of jobs including data exports, report generation, email campaigns, data synchronization, and system maintenance tasks. It integrates seamlessly with the database, object storage, and external APIs.

## Tasks

### Todo: Implement health check endpoint 📅 🔼

`@status: in-progress` `@due: 2026-01-30` `@priority: medium`

Add HTTP endpoint for health monitoring to enable better observability and integration with orchestration tools.

### Todo: Add metrics collection 🔼

`@status: todo` `@priority: medium`

Implement Prometheus metrics for job processing times, success rates, and queue depths.

### Todo: Optimize database connection pooling ⏱️ ❗️

`@status: in-progress` `@priority: high`

Review and optimize connection pool settings to reduce database load during peak processing times.

### Todo: Implement job prioritization 📝

`@status: todo` `@priority: medium`

Add priority levels to jobs so that critical tasks are processed before lower-priority ones.

### Todo: Add integration tests 🔼

`@status: todo` `@priority: medium`

Create comprehensive integration test suite covering main job processing scenarios.

## Checklist: Deployment

1. Update version number in configuration
2. Run unit tests: `go test ./...`
3. Build binary: `make build-runner`
4. Create Docker image: `docker build -t runner:latest .`
5. Tag image with version: `docker tag runner:latest runner:v1.2.3`
6. Push to registry: `docker push runner:v1.2.3`
7. Update Kubernetes manifests with new version
8. Apply rolling update: `kubectl apply -f k8s/runner-deployment.yaml`
9. Monitor logs: `kubectl logs -f deployment/runner`
10. Verify metrics in Grafana dashboard
11. Check job processing rate and error rate
12. Update deployment documentation

## Cheatsheet: View Active Jobs

To view currently running jobs:

```bash
# Connect to Runner pod
kubectl exec -it runner-xyz -- bash

# Query active jobs from database
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
  SELECT id, job_type, status, started_at, updated_at 
  FROM jobs 
  WHERE status = 'running' 
  ORDER BY started_at DESC 
  LIMIT 20;"
```

## Cheatsheet: Restart Stuck Job

If a job is stuck and needs to be restarted:

```bash
# Find the job ID from logs or database
JOB_ID=12345

# Update job status to retry
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
  UPDATE jobs 
  SET status = 'pending', 
      retry_count = retry_count + 1,
      updated_at = NOW()
  WHERE id = $JOB_ID;"

# Monitor job processing
kubectl logs -f deployment/runner | grep "job_id=$JOB_ID"
```

## Cheatsheet: Scale Worker Pool

To adjust the number of concurrent workers:

```bash
# Edit configuration
kubectl edit configmap runner-config

# Update worker_pool_size value (e.g., from 10 to 20)
# Save and exit

# Restart deployment to apply changes
kubectl rollout restart deployment/runner

# Verify new pod count
kubectl get pods -l app=runner

# Monitor performance
kubectl top pods -l app=runner
```

## Cheatsheet: Emergency Stop Processing

To temporarily halt all job processing:

```bash
# Scale down to zero replicas
kubectl scale deployment/runner --replicas=0

# Verify pods are terminated
kubectl get pods -l app=runner

# Jobs will remain in queue and resume when deployment is scaled back up
# To resume:
kubectl scale deployment/runner --replicas=3
```
