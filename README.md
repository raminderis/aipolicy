# aipolicy

A lightweight policy management service for controlling AI agent access to MCP (Model Context Protocol) resources. It provides a REST API to create, update, delete, and evaluate access policies backed by PostgreSQL.

## Overview

aipolicy lets you define named policies that govern whether an AI agent (or any caller) is allowed to access a specific remote MCP service and resource. Each policy can optionally include conditions such as a **time window**, restricting access to certain hours of the day in a given timezone.

## API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/policies` | Create a new policy |
| `GET` | `/policies/{id}` | Retrieve a policy by UUID |
| `PUT` | `/policies/{id}` | Update an existing policy (partial) |
| `DELETE` | `/policies/{id}` | Delete a policy |
| `POST` | `/decide` | Evaluate whether access is allowed |

### Policy fields

| Field | Type | Description |
|-------|------|-------------|
| `policy_id` | string | Human-readable identifier |
| `name` | string | Display name |
| `remote_mcp_service` | string | Target MCP service |
| `resource_access_request` | string | Resource being requested |
| `environment` | string | e.g. `production`, `staging` |
| `enabled` | bool | Whether the policy is active |
| `priority` | int | Evaluation order (lower = higher priority) |
| `description` | string | Optional description |
| `conditions.time_window` | object | Optional time-based condition |

### Time window condition

```json
{
  "conditions": {
    "time_window": {
      "timezone": "America/New_York",
      "start_hour": 9,
      "end_hour": 17
    }
  }
}
```

### Decide request/response

**Request:**
```json
{
  "remote_mcp_service": "my-mcp-service",
  "resource_access_request": "read_file",
  "environment": "production"
}
```

**Response:**
```json
{ "allowed": true }
```

## Running with Docker

```bash
cp .env.example .env   # configure DB connection
docker compose up -d
```

The service listens on port `9099` by default.

## Configuration

Set these environment variables (or use a `.env` file):

| Variable | Default | Description |
|----------|---------|-------------|
| `POLICYMGR_LISTENING_PORT` | `8081` | HTTP port |
| `DBHOST` | — | PostgreSQL host |
| `DBPORT` | — | PostgreSQL port |
| `DBUSER` | — | PostgreSQL user |
| `DBPASSWORD` | — | PostgreSQL password |
| `DBNAME` | — | PostgreSQL database name |

## Tech stack

- **Go** 1.26
- **chi** — HTTP router
- **pgx** — PostgreSQL driver
- **godotenv** — `.env` file loading
