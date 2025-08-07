# Regression Tests

- Live script: `regression-test-backend/run.sh`
  - Loads `.env`, builds server, runs integration tests, shuts server down
- Endpoints covered:
  - /health, /test/llm, /test/structured, /test/models
- Next:
  - Add negative cases and rate-limit handling checks
  - CI integration with masked secrets
