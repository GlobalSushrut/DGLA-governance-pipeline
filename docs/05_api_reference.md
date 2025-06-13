# DGLA API Reference

## Overview

The DGLA API Server provides a comprehensive set of endpoints for authentication, verification, audit logging, and compliance reporting. All endpoints implement cryptographic verification and maintain an immutable audit trail of all operations.

## Base URL

```
http://{host}:{port}
```

Default port: 8081

## Authentication

### Authentication Endpoints

#### POST /auth/login

Authenticates a user and returns a JWT token.

**Request:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "token": "string",
  "user_id": "string",
  "expiration": "number"
}
```

**Status Codes:**
- 200: Success
- 401: Authentication failed
- 429: Rate limit exceeded

#### POST /auth/zk-login

Zero-knowledge authentication that doesn't transmit credentials.

**Request:**
```json
{
  "username": "string",
  "zk_proof": {
    "commitment": "string",
    "challenge_response": "string"
  }
}
```

**Response:**
```json
{
  "token": "string",
  "user_id": "string",
  "expiration": "number",
  "proof_id": "string"
}
```

**Status Codes:**
- 200: Success
- 401: Authentication failed
- 429: Rate limit exceeded

#### POST /auth/refresh

Refreshes a valid JWT token.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

**Response:**
```json
{
  "token": "string",
  "user_id": "string",
  "expiration": "number"
}
```

**Status Codes:**
- 200: Success
- 401: Invalid token
- 429: Rate limit exceeded

#### POST /auth/logout

Invalidates a JWT token.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

**Response:**
```json
{
  "message": "string",
  "log_id": "string"
}
```

**Status Codes:**
- 200: Success
- 401: Invalid token

## Verification API

### Verification Endpoints

#### POST /verify/create-proof

Creates a cryptographic proof of data integrity.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Body:
```json
{
  "data": "object"
}
```

**Response:**
```json
{
  "id": "string",
  "hash": "string",
  "timestamp": "number",
  "signature": "string"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid request
- 401: Unauthorized
- 429: Rate limit exceeded

#### GET /verify/validate/{proof_id}

Validates a previously created proof.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

**Response:**
```json
{
  "valid": "boolean",
  "proof_id": "string",
  "timestamp": "number",
  "verification_id": "string"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid proof ID
- 401: Unauthorized
- 404: Proof not found

#### POST /verify/hash

Creates a cryptographic hash of provided data.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Body:
```json
{
  "data": "string",
  "algorithm": "string",
  "params": "object"
}
```

**Response:**
```json
{
  "hash": "string",
  "algorithm": "string"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid request
- 401: Unauthorized
- 429: Rate limit exceeded

## Audit Log API

### Chain Log Endpoints

#### POST /chainlog/append

Appends a new entry to the immutable audit log.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Body:
```json
{
  "entity_id": "string",
  "entity_type": "string",
  "action": "string",
  "metadata": "object"
}
```

**Response:**
```json
{
  "id": "string",
  "timestamp": "number",
  "hash": "string",
  "previous_hash": "string"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid request
- 401: Unauthorized
- 429: Rate limit exceeded

#### GET /chainlog/verify/{entity_id}/{entity_type}

Verifies the integrity of the audit log chain for an entity.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

**Response:**
```json
{
  "verified": "boolean",
  "entity_id": "string",
  "entity_type": "string",
  "log_count": "number",
  "verification_id": "string"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid request
- 401: Unauthorized
- 404: Entity not found

#### GET /chainlog/logs/{entity_id}/{entity_type}

Retrieves audit logs for an entity.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Query Parameters:
- start_time (optional): Starting timestamp
- end_time (optional): Ending timestamp
- limit (optional): Maximum number of records
- offset (optional): Pagination offset

**Response:**
```json
{
  "entity_id": "string",
  "entity_type": "string",
  "logs": [
    {
      "id": "string",
      "timestamp": "number",
      "action": "string",
      "metadata": "object",
      "hash": "string",
      "previous_hash": "string"
    }
  ],
  "count": "number",
  "total": "number"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid request
- 401: Unauthorized
- 404: Entity not found

## Export API

### Export Endpoints

#### POST /export/logs

Exports logs for compliance and audit purposes.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Body:
```json
{
  "entity_id": "string",
  "entity_type": "string",
  "start_time": "number",
  "end_time": "number",
  "format": "string"
}
```

**Response:**
```json
{
  "export_id": "string",
  "timestamp": "number",
  "record_count": "number",
  "download_url": "string",
  "proof_id": "string"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid request
- 401: Unauthorized
- 404: No logs found

#### POST /export/compliance

Generates a compliance report.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Body:
```json
{
  "reportType": "string",
  "startTime": "number",
  "endTime": "number",
  "entityId": "string",
  "format": "string"
}
```

**Response:**
```json
{
  "report_id": "string",
  "timestamp": "number",
  "type": "string",
  "download_url": "string",
  "proof_id": "string"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid request
- 401: Unauthorized
- 404: No data found
- 429: Rate limit exceeded

## Administration API

### System Endpoints

#### GET /system/status

Returns the current system status.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

**Response:**
```json
{
  "status": "string",
  "version": "string",
  "uptime": "number",
  "metrics": {
    "requests_total": "number",
    "proofs_created": "number",
    "logs_appended": "number",
    "active_users": "number"
  },
  "health": {
    "redis": "boolean",
    "storage": "boolean",
    "cpu": "number",
    "memory": "number"
  }
}
```

**Status Codes:**
- 200: Success
- 401: Unauthorized
- 403: Forbidden (insufficient privileges)

#### POST /system/metrics

Returns detailed system metrics with cryptographic validation.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Body:
```json
{
  "metrics": ["string"],
  "start_time": "number",
  "end_time": "number"
}
```

**Response:**
```json
{
  "timestamp": "number",
  "metrics": "object",
  "proof_id": "string"
}
```

**Status Codes:**
- 200: Success
- 400: Invalid request
- 401: Unauthorized
- 403: Forbidden (insufficient privileges)

## User Management API

### User Endpoints

#### POST /users/create

Creates a new user.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Body:
```json
{
  "username": "string",
  "password": "string",
  "role": "string",
  "metadata": "object"
}
```

**Response:**
```json
{
  "user_id": "string",
  "username": "string",
  "created_at": "number",
  "role": "string",
  "proof_id": "string"
}
```

**Status Codes:**
- 201: Created
- 400: Invalid request
- 401: Unauthorized
- 403: Forbidden (insufficient privileges)
- 409: Username already exists

#### GET /users/{user_id}

Retrieves user information.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

**Response:**
```json
{
  "user_id": "string",
  "username": "string",
  "created_at": "number",
  "last_login": "number",
  "role": "string",
  "metadata": "object"
}
```

**Status Codes:**
- 200: Success
- 401: Unauthorized
- 403: Forbidden (insufficient privileges)
- 404: User not found

## Role Management API

### Role Endpoints

#### POST /roles/create

Creates a new role.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

Body:
```json
{
  "name": "string",
  "permissions": ["string"],
  "metadata": "object"
}
```

**Response:**
```json
{
  "role_id": "string",
  "name": "string",
  "permissions": ["string"],
  "created_at": "number",
  "proof_id": "string"
}
```

**Status Codes:**
- 201: Created
- 400: Invalid request
- 401: Unauthorized
- 403: Forbidden (insufficient privileges)
- 409: Role already exists

#### GET /roles/{role_id}

Retrieves role information.

**Request:**
Headers:
```
Authorization: Bearer {token}
```

**Response:**
```json
{
  "role_id": "string",
  "name": "string",
  "permissions": ["string"],
  "created_at": "number",
  "metadata": "object"
}
```

**Status Codes:**
- 200: Success
- 401: Unauthorized
- 403: Forbidden (insufficient privileges)
- 404: Role not found

## Error Responses

All API errors follow a consistent format:

```json
{
  "error": {
    "code": "string",
    "message": "string",
    "details": "object"
  }
}
```

## Rate Limiting

All endpoints are protected by cryptographically verified rate limiting:

- Limits are set per endpoint and user role
- Rate limit responses include proof of rate limit decision
- Limits are configurable via system configuration

## Authentication Headers

All authenticated endpoints require:

```
Authorization: Bearer {token}
```

## Content Types

Unless otherwise specified:

- Request Content-Type: application/json
- Response Content-Type: application/json

## Versioning

The API version is specified in the URL:

```
/v1/resource
```

If not specified, the latest version is used.

## Conclusion

This API reference provides a comprehensive overview of the DGLA API endpoints. All endpoints implement cryptographic verification and maintain an immutable audit trail of operations, providing 1000x stronger security guarantees than traditional API approaches.
