# NextGen IGA: Time-Bound Access Management (TBAM) Microservice

## 📖 Overview
The Time-Bound Access Management (TBAM) microservice is a high-performance backend engine built in Go. It acts as a core component of the NextGen Identity Governance and Administration (IGA) platform. This service handles the automated provisioning and precise, time-bound deprovisioning of highly privileged access within an LDAP directory.

Designed for fault-tolerance and high concurrency, TBAM utilizes an Event-Driven Architecture via NATS JetStream to ingest requests, Go's native in-memory `time.AfterFunc` heaps for exact-second revocations, and MySQL to maintain a strict, immutable audit trail.

## Architectural Diagram
![Architecture](images/architecture.png)

## 🚀 Key Features
* **1:N Multi-Group Scheduling:** A single user can hold multiple temporary access grants to different privilege groups simultaneously, each with its own independent expiration timer.
* **In-Memory Revocation Engine:** Utilizes Go's lightweight timer heap to trigger LDAP revocations at the exact second of expiration without database contention, ensuring access is automatically revoked upon the expiry date.
* **Boot-Up State Recovery:** Automatically scans the LDAP directory on startup to rebuild its in-memory timers, guaranteeing zero "lingering access" in the event of a server restart.
* **Event-Driven Ingestion:** Uses NATS JetStream for durable message queuing. HTTP API requests are instantly persisted to the event bus and processed asynchronously by background workers.
* **Immutable Audit Trail:** Every successful or failed provisioning and deprovisioning event is logged directly to a MySQL `access_audit_logs` table, supporting downstream AI-chatbot reporting capabilities.
* **High-Concurrency Resource Pools:** Implements custom connection pooling for LDAP and MySQL, alongside a bounded Goroutine worker pool, protecting upstream directories from traffic spikes.

## 🛠️ Tech Stack
* **Language:** Go (Golang)
* **Web Framework:** Gin (HTTP API)
* **Message Broker:** NATS JetStream
* **Directory Server:** Apache Directory Studio (LDAP / OpenLDAP)
* **Audit Database:** MySQL

## 📂 Project Structure
```text
iga-timebound/
├── cmd/
│   └── server/
│       └── main.go           # App entry point (Boot-up recovery, API, NATS Listener)
├── internal/
│   ├── api/
│   │   └── handlers.go       # Gin HTTP endpoints and NATS Subscription logic
│   ├── db/
│   │   └── mysql.go          # MySQL connection pooling and Audit Logging
│   ├── ldap/
│   │   ├── client.go         # Thread-safe LDAP connection pooling
│   │   ├── provision.go      # Group assignment, timer attachment, and DB logging
│   │   ├── search.go         # Boot-up query to find active active timers
│   │   └── modify.go         # Targeted group removal, timer cleanup, and DB logging
│   ├── models/
│   │   ├── grant.go          # Data structure for 1:N access grants
│   │   └── request.go        # JSON schema for incoming API requests
│   ├── scheduler/
│   │   └── scheduler.go      # In-memory time.AfterFunc management
│   └── worker/
│       └── pool.go           # Concurrency management and task queuing
├── .env                      # Local environment configurations
├── go.mod                    # Go dependencies
└── go.sum                    # Go checksums
```

## ⚙️ Setup & Installation
### 1. Prerequisites
- Go 1.20+
- NATS Server (with JetStream enabled)
- MySQL Server
- LDAP Server (e.g., Apache Directory Studio / OpenLDAP)

### 2. Environment Variables
Create a `.env` file in the root directory:

```ini
# LDAP Configuration
LDAP_URL=ldap://localhost:3890
LDAP_BIND_DN=cn=admin,dc=example,dc=com
LDAP_PASSWORD=admin
LDAP_BASE_DN=dc=example,dc=com

# NATS Configuration
NATS_URL=nats://localhost:4222

# MySQL Configuration
DB_USER=root
DB_PASSWORD=secret
DB_HOST=localhost
DB_PORT=3306
DB_NAME=iga_audit

# Worker Pool Configuration
WORKER_COUNT=20
```

### 3. Database Migration
Ensure your MySQL database has the required audit table:

```sql
CREATE TABLE access_audit_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    action_type VARCHAR(50) NOT NULL,
    target_uid VARCHAR(100) NOT NULL,
    target_group VARCHAR(255) NOT NULL,
    expiry_time DATETIME,
    status VARCHAR(20) NOT NULL,
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 4. Running the Service
To start the microservice:

```bash
go run main.go
```

The service will connect to all data stores, perform its boot-up recovery scan, initialize the NATS listener, and start the Gin server on `0.0.0.0:8080`.

## 🔌 API Documentation
### Provision Time-Bound Access
Persists a provisioning request to NATS JetStream for asynchronous processing.
**Endpoint:** `POST /api/access/time`
**Content-Type:** `application/json`
**Request Payload:**
```json
{
  "uid": "jdoe_123",
  "privilege_access": "PrivilegedAdmins",
  "end_date": "2024-05-15",
  "end_time": "17:00:00"
}
```

**Response (200 OK):**
```json
{
  "message": "Request persisted in JetStream"
}
```

## 🧠 Core System Workflows
### 1. The Provisioning Saga
1. **Ingest:** HTTP request hits the Gin API. The payload is validated and published to the `events.provision.time` NATS stream.
2. **Consume:** The background NATS Listener pulls the event and submits it to the Worker Pool.
3. **Execute:** A worker borrows an LDAP connection, adds the user to the `uniqueMember` list of the target group, and appends a `"GroupDN|Timestamp"` string to the user's `businessCategory` array.
4. **Audit & Schedule:** The worker logs a `GRANT` event to MySQL and passes the grant to the Scheduler to set the in-memory timer.

### 2. The Deprovisioning Saga
1. **Trigger:** The in-memory `time.AfterFunc` alarm fires.
2. **Execute:** The LDAP client surgically removes the user from the specific `uniqueMember` group and deletes the specific `"GroupDN|Timestamp"` string from their `businessCategory` attribute.
3. **Audit:** A `REVOKE` (or `REVOKE_FAILED`) event is logged to MySQL.

### 3. Boot-Up Recovery
To prevent amnesia during server restarts, `main.go` executes `FetchExpiringGrants()` before initializing the API. It searches the LDAP directory for any user possessing the `businessCategory` attribute, splits the stored strings, recalculates the time remaining from "Now", and reinstates all in-memory timers.
