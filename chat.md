
## Step 2 — Define users, tenants, roles & core workflows

Before touching architecture or code, we need to define **who uses the system and what they actually do**.

### 1. Tenant model

The fundamental boundary is the **Agency/Organization**.

```text
Platform
│
├── Agency A
│   ├── Users
│   ├── Clients
│   ├── Projects
│   ├── Tasks
│   ├── Time entries
│   ├── Invoices
│   └── Files
│
├── Agency B
│   ├── Users
│   ├── Clients
│   ├── Projects
│   └── ...
│
└── Agency C
```

Every piece of business data belongs to an agency.

For example:

```text
projects.agency_id
clients.agency_id
tasks.agency_id
invoices.agency_id
```

This gives us a clean multi-tenant isolation model.

**Important:** we won't create a separate database/schema for every agency. We'll start with a shared PostgreSQL database with strong `agency_id` boundaries.

---

# 2. Users

We'll have two fundamentally different types of people interacting with the platform:

### Internal users

People who work for the agency:

* Owner
* Admin
* Manager
* Employee

### External users

People belonging to the agency's clients:

* Client

So conceptually:

```text
Agency
  │
  ├── Internal Users
  │     ├── Owner
  │     ├── Admin
  │     ├── Manager
  │     └── Employee
  │
  └── Client Users
        └── Client
```

---

# 3. Roles

I'd start with **RBAC** rather than trying to build some complicated permission engine.

### Owner

Full control over the agency.

Can:

* manage agency settings
* manage users
* manage billing
* manage clients
* manage projects
* manage finances
* view reports
* manage integrations

### Admin

Operational administrator.

Can:

* manage users
* manage clients
* manage projects
* manage tasks
* manage invoices
* manage files
* view reports

But cannot manage the agency's subscription/billing unless explicitly granted later.

### Manager

Responsible for projects and teams.

Can:

* create/manage projects
* assign employees
* manage tasks
* manage project budgets
* view time tracking
* communicate with clients
* view project reports

### Employee

Day-to-day worker.

Can:

* view assigned projects
* manage assigned tasks
* log time
* upload files
* communicate
* update task status

### Client

External portal user.

Can only see things belonging to their client organization:

* projects
* project progress
* tasks they're allowed to see
* files
* invoices
* messages
* approvals

They **cannot** see internal agency information.

---

# 4. Client model

A really important distinction:

**Agency ≠ Client ≠ Client User**

For example:

```text
Acme Digital Agency
│
├── Users
│   ├── John - Owner
│   ├── Sarah - Manager
│   └── Mike - Developer
│
├── Client: Coca-Cola
│   ├── User: Alice
│   └── User: Bob
│
└── Client: Nike
    └── User: Charlie
```

A `Client` is a business/customer of the agency.

A `ClientUser` is an actual person working for that client.

This distinction will become extremely useful later for:

* client portals
* permissions
* project access
* communication
* approvals
* billing

---

# 5. Core business workflow

This is the heart of the product.

I'd define the initial workflow as:

```text
Agency
   │
   ▼
Client
   │
   ▼
Project
   │
   ├──────────────┐
   ▼              ▼
Tasks          Team Members
   │
   ▼
Time Tracking
   │
   ▼
Project Completion
   │
   ▼
Invoice
   │
   ▼
Payment
```

But there's another important workflow:

```text
Client
   │
   ▼
Communication
   │
   ├── Messages
   ├── Files
   ├── Comments
   └── Approvals
```

---

# 6. The major domains

This gives us the initial domain map.

```text
┌─────────────────────────────────────────┐
│              AGENCY                     │
│                                         │
│  Users / Roles / Settings / Billing     │
└────────────────┬────────────────────────┘
                 │
        ┌────────┴────────┐
        ▼                 ▼
     CLIENTS           PROJECTS
                          │
                 ┌────────┼─────────┐
                 ▼        ▼         ▼
               TASKS     TEAM      FILES
                         │
                         ▼
                       TIME
                         │
                         ▼
                      REPORTS
                         
        CLIENTS
           │
           ├── Contacts
           ├── Projects
           ├── Files
           ├── Messages
           └── Invoices
```

Eventually we'll probably have domains around:

1. **Identity & Authentication**
2. **Agency**
3. **Users & Roles**
4. **Clients**
5. **Projects**
6. **Tasks**
7. **Teams**
8. **Time Tracking**
9. **Files**
10. **Communication**
11. **Invoicing**
12. **Payments**
13. **Reporting**
14. **Notifications**
15. **Audit Logs**
16. **Subscriptions/Billing**
17. **Integrations**

But **we are not building all of these initially**.

---

# 7. MVP workflow

For the first usable version, I'd deliberately reduce it to:

```text
Authentication
      │
      ▼
Agency
      │
      ├── Employees
      │
      └── Clients
              │
              ▼
           Projects
              │
              ▼
            Tasks
              │
              ▼
         Time Tracking
              │
              ▼
            Files
```

And then:

```text
Project
   │
   ├── Tasks
   ├── Members
   ├── Time
   ├── Files
   └── Client access
```

That is already a legitimate SaaS.

Then we can progressively add:

```text
MVP
 ↓
Communication
 ↓
Invoicing
 ↓
Payments
 ↓
Reporting
 ↓
Automation
 ↓
Integrations
 ↓
AI features
```

This gives us a project that can grow substantially without having to redesign the foundation.

---

# 8. One important architectural decision

I want us to treat **Project** as one of the central aggregates of the system.

For example:

```text
Client
  │
  └── Project
        │
        ├── Project Members
        ├── Tasks
        ├── Time Entries
        ├── Files
        ├── Comments
        ├── Activity
        └── Invoices
```

This makes the platform useful for many agency types:

* software agencies
* marketing agencies
* design agencies
* SEO agencies
* consulting firms
* creative agencies

without hardcoding the product around one specific industry.

---

## Step 2 result

So our initial domain is now:

**Tenant**

```text
Agency
```

**Internal actors**

```text
Owner
Admin
Manager
Employee
```

**External actors**

```text
Client User
```

**Core entities**

```text
Agency
User
Role
Client
Client User
Project
Project Member
Task
Time Entry
File
```

**Primary workflow**

```text
Agency
  ↓
Employees + Clients
  ↓
Projects
  ↓
Tasks
  ↓
Time Tracking
  ↓
Files / Communication
  ↓
Invoicing / Payments
```

And the architectural principle is:

> **Multi-tenant modular monolith, with the Agency as the tenant boundary and RBAC for authorization.**

That's enough for Step 2. **Next should be Step 3: turn this domain into an actual system design — defining the boundaries between modules and how the Go backend, PostgreSQL, Redis, background workers, React frontend, and external services communicate.**
Yes. Now we're at **Step 3: System Design**.

The goal here is to decide **how the pieces fit together**, before we choose exact libraries, versions, folder structures, or start coding.

# Step 3 — System Design

## 3.1 High-level architecture

I recommend this:

```text
                         INTERNET
                            │
                            ▼
                  ┌──────────────────┐
                  │ Reverse Proxy    │
                  │ TLS / Routing    │
                  └────────┬─────────┘
                           │
              ┌────────────┴────────────┐
              │                         │
              ▼                         ▼
       React Web App               Go REST API
       Static frontend             Application
                                        │
                         ┌──────────────┼──────────────┐
                         │              │              │
                         ▼              ▼              ▼
                    PostgreSQL       Redis         Worker
                    source of truth  cache/queue   async jobs
                         │              │              │
                         └──────────────┼──────────────┘
                                        │
                                        ▼
                              External Services
                              ├── Email
                              ├── Object Storage
                              ├── Payments
                              └── OAuth/etc.
```

The important part is that **we are not starting with microservices**.

We're building a **modular monolith**.

---

# 3.2 Why modular monolith?

Our Go backend is one deployable application:

```text
gorest-api
```

But internally it's divided into clear business modules:

```text
auth
agency
users
clients
projects
tasks
time
files
invoices
notifications
...
```

So:

```text
                    Go Application
                         │
       ┌─────────────────┼─────────────────┐
       │                 │                 │
     Clients          Projects           Tasks
       │                 │                 │
       └─────────────────┼─────────────────┘
                         │
                     PostgreSQL
```

This gives us most of the **architectural benefits of services without the operational nightmare of microservices**.

We don't need:

* service discovery
* Kubernetes
* inter-service networking
* distributed transactions
* dozens of Docker containers
* gRPC everywhere
* distributed tracing just to understand a simple request

But we still establish boundaries so that if one module eventually needs to become a separate service, we can do that.

---

# 3.3 Backend modules

Our initial Go application will conceptually look like:

```text
internal/
├── auth/
├── agency/
├── user/
├── client/
├── project/
├── task/
├── time/
├── file/
├── notification/
└── ...
```

Each module owns its business logic.

For example:

```text
project/
├── handler
├── service
├── repository
└── model
```

We'll define the **actual folder structure later**.

I'm deliberately not locking us into a specific implementation yet.

---

# 3.4 Request flow

Consider:

> Manager opens a project.

The request goes:

```text
Browser
   │
   │ GET /api/projects/{id}
   ▼
Reverse Proxy
   │
   ▼
Go HTTP Server
   │
   ▼
Auth Middleware
   │
   ▼
Project Handler
   │
   ▼
Project Service
   │
   ├── authorization
   │
   └── business rules
   │
   ▼
Project Repository
   │
   ▼
PostgreSQL
   │
   ▼
Repository
   │
   ▼
Service
   │
   ▼
Handler
   │
   ▼
JSON Response
   │
   ▼
React
```

That's the basic synchronous request path.

---

# 3.5 PostgreSQL

**PostgreSQL is our primary source of truth.**

Everything important goes there.

For example:

```text
agencies
users
agency_members
clients
client_users
projects
project_members
tasks
time_entries
files
invoices
payments
notifications
audit_logs
...
```

We don't use Redis as a second database.

If Redis disappears:

```text
Redis 💥
```

the application should still have all critical business data in PostgreSQL.

Redis can be rebuilt.

---

# 3.6 Multi-tenancy

This is one of the most important architectural decisions.

Every tenant-owned resource has an agency relationship.

For example:

```text
agencies
    │
    ├── users/members
    ├── clients
    ├── projects
    ├── tasks
    ├── invoices
    └── ...
```

A request carries the authenticated user's agency context.

Conceptually:

```text
JWT
 │
 ├── user_id
 └── agency_id
```

Then:

```text
GET /projects/123
```

doesn't simply mean:

```sql
SELECT * FROM projects WHERE id = $1;
```

It means something closer to:

```sql
SELECT *
FROM projects
WHERE id = $1
  AND agency_id = $2;
```

This is **critical**.

A user from:

```text
Agency A
```

must never be able to retrieve:

```text
Agency B's project
```

even if they somehow know the project's UUID.

We'll reinforce this at the authorization/data-access level rather than trusting the frontend.

---

# 3.7 Redis

Redis will have several potential responsibilities, but we shouldn't use it everywhere just because we're using Redis.

Good candidates:

### Background job coordination

```text
API
 │
 ▼
Redis / Queue
 │
 ▼
Worker
```

### Caching

For data where caching actually provides value.

### Rate limiting

For things like:

```text
POST /auth/login
POST /auth/forgot-password
```

### Temporary data

For example:

```text
email verification
short-lived tokens
job state
```

But:

> **PostgreSQL remains authoritative.**

---

# 3.8 Background worker

Some operations shouldn't block an HTTP request.

For example:

```text
User creates project
        │
        ▼
API saves project
        │
        ▼
Queue job
        │
        ▼
HTTP response ───────────────► User
        │
        │
        ▼
      Worker
        │
        ├── send notification
        ├── send email
        ├── generate document
        └── other async work
```

The user shouldn't have to wait five seconds because we're sending an email.

We'll therefore have:

```text
API
Worker
```

as two processes from the same Go codebase.

Not two microservices.

---

# 3.9 File storage

We should **not store uploaded files directly inside PostgreSQL**.

For example:

* project attachments
* client documents
* invoices
* contracts
* images
* exported reports

Instead:

```text
React
  │
  ▼
Go API
  │
  ▼
Object Storage
```

PostgreSQL stores metadata:

```text
files
------------------------
id
agency_id
project_id
uploaded_by
object_key
filename
content_type
size
created_at
```

The actual bytes live in object storage.

This will also make things like signed URLs and secure downloads straightforward later.

---

# 3.10 Email

Email should also be asynchronous.

For example:

```text
Manager creates invitation
          │
          ▼
PostgreSQL
          │
          ▼
Queue
          │
          ▼
Worker
          │
          ▼
Email provider
```

So our API isn't coupled to an SMTP request during the user's HTTP request.

---

# 3.11 Frontend architecture

The React application communicates with the Go API:

```text
React
  │
  │ HTTPS / JSON
  ▼
Go REST API
```

We don't need GraphQL initially.

REST is a very good fit here because our domain has clear resources:

```text
/api/auth
/api/users
/api/clients
/api/projects
/api/tasks
/api/time-entries
/api/files
/api/invoices
```

React will primarily have:

### Server state

Handled by TanStack Query.

```text
clients
projects
tasks
invoices
```

### Local/UI state

Handled locally or with Zustand where appropriate.

For example:

```text
sidebar state
modal state
filters
UI preferences
```

We should **not put all server data into Zustand**.

---

# 3.12 Authentication

Initial architecture:

```text
React
   │
   ▼
Go API
   │
   ├── Login
   ├── Access token
   └── Refresh token
```

We'll use short-lived access tokens and refresh tokens.

The important thing is that authentication and authorization are separate concepts:

```text
Authentication
"Who are you?"

        ↓

Authorization
"What are you allowed to do?"
```

For example:

```text
User = John
Agency = Acme
Role = Manager
```

Then authorization determines:

```text
Can John edit this project?
Can John see this invoice?
Can John manage employees?
Can John access this client?
```

---

# 3.13 Authorization model

Initially:

**RBAC + resource ownership/membership checks.**

Example:

```text
Role:
Manager
```

doesn't automatically mean:

```text
Manager can access every resource in the database.
```

We may also need:

```text
Project membership
Client relationship
Agency ownership
```

So authorization can look like:

```text
                 Request
                    │
                    ▼
              Authenticated?
                    │
                    ▼
                 Agency?
                    │
                    ▼
                  Role?
                    │
                    ▼
           Resource relationship?
                    │
                    ▼
                  ALLOW
```

This is much more realistic than simply having:

```text
if role == "admin" { ... }
```

everywhere.

---

# 3.14 Observability

From the beginning, the application should produce:

### Logs

Structured JSON logs:

```text
timestamp
level
request_id
user_id
agency_id
method
path
status
duration
```

### Metrics

Things such as:

```text
HTTP request count
HTTP latency
HTTP errors
DB connection pool
queue depth
worker failures
```

### Traces

Eventually:

```text
HTTP request
   │
   ├── PostgreSQL
   │
   ├── Redis
   │
   └── external API
```

This becomes extremely useful when debugging production issues.

We don't need to build the entire observability stack today, but **the architecture should support it from day one.**

---

# 3.15 Deployment architecture

Initially, on the Linode VPS:

```text
                    INTERNET
                       │
                       ▼
                 ┌───────────┐
                 │  Reverse  │
                 │   Proxy   │
                 └─────┬─────┘
                       │
          ┌────────────┴────────────┐
          │                         │
          ▼                         ▼
      React/Nginx                 Go API
                                    │
                      ┌─────────────┼────────────┐
                      │             │            │
                      ▼             ▼            ▼
                 PostgreSQL       Redis       Worker
```

Everything can initially run through Docker Compose.

Something conceptually like:

```text
docker compose
│
├── reverse-proxy
├── frontend
├── api
├── worker
├── postgres
└── redis
```

We can later move to Kubernetes if there is a real reason.

---

# 3.16 CI/CD

Eventually:

```text
Developer
   │
   ▼
Git Push / PR
   │
   ▼
GitHub Actions
   │
   ├── format
   ├── lint
   ├── typecheck
   ├── unit tests
   ├── integration tests
   ├── security checks
   └── build
          │
          ▼
     Docker image
          │
          ▼
       Registry
          │
          ▼
       Linode
          │
          ▼
    Pull new image
          │
          ▼
    DB migrations
          │
          ▼
    Health checks
          │
          ▼
      Production
```

We'll design this properly later.

---

# 3.17 The complete system

Putting everything together:

```text
                           USERS
                             │
                             ▼
                    ┌─────────────────┐
                    │  React Web App  │
                    └────────┬────────┘
                             │ HTTPS
                             ▼
                    ┌─────────────────┐
                    │ Reverse Proxy   │
                    │ TLS / Routing   │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    Go API       │
                    │                 │
                    │ ┌─────────────┐ │
                    │ │ Auth        │ │
                    │ │ Agency      │ │
                    │ │ Users       │ │
                    │ │ Clients     │ │
                    │ │ Projects    │ │
                    │ │ Tasks       │ │
                    │ │ Time        │ │
                    │ │ Files       │ │
                    │ │ Invoices    │ │
                    │ └─────────────┘ │
                    └───────┬─────────┘
                            │
              ┌─────────────┼──────────────┐
              │             │              │
              ▼             ▼              ▼
        ┌──────────┐   ┌──────────┐   ┌──────────┐
        │PostgreSQL│   │  Redis   │   │  Worker  │
        │          │   │          │   │          │
        │ Source   │   │ Cache    │   │ Async    │
        │ of Truth │   │ Queue    │   │ Jobs     │
        └──────────┘   └──────────┘   └────┬─────┘
                                            │
                              ┌─────────────┼─────────────┐
                              ▼             ▼             ▼
                           Email        Storage       Payments
```

## The key decisions we've made

| Decision         | Choice                                |
| ---------------- | ------------------------------------- |
| Architecture     | **Modular monolith**                  |
| Backend          | **Go**                                |
| Frontend         | **React + TypeScript**                |
| API              | **REST**                              |
| Primary DB       | **PostgreSQL**                        |
| Cache/queue      | **Redis**                             |
| Async processing | **Go worker**                         |
| File storage     | **Object storage**                    |
| Multi-tenancy    | **Shared DB + `agency_id` isolation** |
| Authorization    | **RBAC + resource checks**            |
| Deployment       | **Docker**                            |
| Initial hosting  | **Linode VPS**                        |
| Reverse proxy    | TBD                                   |
| CI/CD            | **GitHub Actions**                    |
| Observability    | **Logs + metrics + traces**           |
| Kubernetes       | **Not initially**                     |

**This is the system design I would actually choose for the project.** It is sophisticated enough to demonstrate real production engineering, but we're not adding infrastructure just for the sake of saying we used it.
