# NextGen IGA: Time-Bound Access Management (TBAM) Microservice

## 📖 Overview
The Time-Bound Access Management (TBAM) microservice is a high-performance backend engine built in Go. It is a core component of the NextGen Identity Governance and Administration (IGA) platform. This service handles the automated provisioning and precise, time-bound deprovisioning of highly privileged access within an LDAP directory.

Instead of relying on heavy database polling, TBAM utilizes Go's native concurrency and in-memory `time.AfterFunc` heaps to schedule precise access revocations with an extremely low memory footprint.

## 🚀 Key Features
* **1:N Multi-Group Scheduling:** A single user can hold multiple temporary access grants to different groups simultaneously, each with its own independent expiration timer.
* **In-Memory Revocation Engine:** Utilizes Go's lightweight timer heap to trigger LDAP revocations at the exact second of expiration without database contention.
* **Boot-Up State Recovery:** Automatically scans the LDAP directory on startup to rebuild its in-memory timers, ensuring zero "lingering access" if the server restarts.
* **High-Concurrency Worker Pools:** Safely processes massive bursts of API provisioning requests through a buffered worker queue, protecting the directory server from load spikes.
* **Thread-Safe LDAP Connection Pooling:** Maintains a persistent pool of LDAP connections shared concurrently across the API and the Scheduler.

## 🛠️ Tech Stack
* **Language:** Go (Golang)
* **Web Framework:** Gin (HTTP API)
* **Directory Server:** Apache Directory Studio (LDAP / OpenLDAP)
* **Event Broker:** NATS (Integration Pending)

## 📂 Project Structure
```text
iga-timebound/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point (Starts API and Scheduler)
├── internal/
│   ├── api/
│   │   └── handlers.go       # Gin HTTP routing and endpoint logic
│   ├── ldap/
│   │   ├── client.go         # Connection pooling and LDAP binding
│   │   ├── provision.go      # Group assignment and timer attribute attachment
│   │   ├── search.go         # Boot-up queries to find active timers
│   │   └── modify.go         # Targeted group removal and timer cleanup
│   ├── models/
│   │   ├── grant.go          # Data structures for 1:N access grants
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
- Docker (for local OpenLDAP testing)
- Apache Directory Studio

### 2. Environment Variables
Create a `.env` file in the root directory (do not commit this to version control):
```
LDAP_URL=ldap://localhost:3890
LDAP_BIND_DN=cn=admin,dc=example,dc=com
LDAP_PASSWORD=admin
LDAP_BASE_DN=dc=example,dc=com
```

### 3. Running the Service
To start the microservice locally:
```bash
cd cmd/server
go run main.go
```

The service will perform its boot-up recovery scan and then start the Gin server on port `5000`.

## 🔌 API Documentation

### Provision Time-Bound Access
Assigns a user to a privileged LDAP group and schedules their automated removal.

**Endpoint:** `POST /provision`
**Content-Type:** `application/json`
**Request Payload:**
```json
{
  "uid": "jdoe_123",
  "grp_associated": "BaseEmployees",
  "privilege_access": "PrivilegedAdmins",
  "end_date": "2024-05-15",
  "end_time": "17:00:00"
}
```

**Response (202 Accepted):**
```json
{
  "status": "queued",
  "user": "jdoe_123"
}
```

## 🧠 Core Architecture Details

### The "Access Grant" Data Model
To allow a user to hold multiple temporary access grants, the engine hijacks a multi-valued LDAP attribute (e.g., `businessCategory`). When access is provisioned, the Go service appends a concatenated string formatted as `GroupDN|UnixTimestamp`.

Example of a user's `businessCategory` array in LDAP:
- `cn=DB_Admins,ou=Groups,dc=example,dc=com|1714000000`
- `cn=Server_Admins,ou=Groups,dc=example,dc=com|1715000000`

### The Execution Lifecycle
1. **Ingestion:** A POST request hits the Gin API.
2. **Delegation:** The API validates the JSON and drops the task into the background Worker Pool.
3. **Provisioning:** A worker borrows an LDAP connection, adds the user to the target group, and appends the `GroupDN|Timestamp` string to the user.
4. **Scheduling:** The worker immediately passes the new grant to the Scheduler.
5. **Waiting:** The Scheduler calculates the remaining time and sets a lightweight `time.AfterFunc` alarm in Go's memory heap.
6. **Deprovisioning:** At the exact second of expiration, the alarm triggers a new goroutine. This routine surgically removes the user from the specific group and deletes the specific `GroupDN|Timestamp` string from their profile.
