# **Aetherium: An Architectural Blueprint for Secure, Autonomous, Containerized AI Agent Systems**

## **Executive Summary**

This report presents the architectural design for "Aetherium," an autonomous, containerized AI agent platform engineered for security, auditability, and a developer-centric workflow. The system is designed to execute AI-driven tasks, such as code generation or analysis using Large Language Model (LLM) command-line interfaces (CLIs) like Claude and Gemini, in a fully automated "YOLO mode." The core value proposition of Aetherium lies in its ability to provide a secure, sandboxed environment for these agents, mitigating the inherent risks of running autonomous code while integrating seamlessly into modern software development practices.

The architecture is founded upon several key design pillars. First is a strict separation of concerns, realized through a bifurcated **Control Plane** and **Execution Plane**. This paradigm, borrowed from robust networking and distributed systems design, ensures that system management and orchestration are physically and logically isolated from the agent workloads, providing a powerful defense against system-wide compromise. Second is a novel, **GitHub-centric feedback loop**, where the primary output of any agent task is a Pull Request (PR), and subsequent interactions are driven by comments on that PR. This transforms a familiar developer tool into an intuitive and auditable control surface for iterative AI tasks. Third is a commitment to **comprehensive observability**, achieved through centralized logging and real-time status monitoring, granting operators full visibility and control over agent lifecycles.

For its initial implementation, Aetherium will leverage Docker containers, hardened with a defense-in-depth security posture. However, the architecture is designed with a strategic evolution path towards **microVM-based runtimes**, such as Kata Containers, to achieve hardware-level isolation. This forward-looking approach ensures the system can meet the escalating security demands of running increasingly powerful and untrusted AI agents in production environments. This document serves as an exhaustive blueprint for constructing Aetherium, from its initial MVP to a production-ready state.

## **Section 1: Foundational System Architecture**

The macro-level design of Aetherium is predicated on core principles of isolation and separation of concerns. This approach is the most critical architectural decision for meeting the dual requirements of operational safety and robust manageability. By logically and physically decoupling the system's management functions from its workload execution, a resilient and secure foundation is established.

### **1.1 The Control Plane vs. Execution Plane Paradigm: A Non-Negotiable Foundation for Security**

The fundamental architectural pattern for Aetherium is the separation of the system into two distinct planes: a Control Plane and an Execution Plane. This design is not merely a suggestion but a foundational requirement, drawing from decades of best practices in building secure, scalable, and compliant distributed systems.1

* **The Control Plane:** This is the authoritative "brain" of the system, responsible for all management, orchestration, configuration, and metadata handling.1 It exposes the user-facing API, processes task requests, manages the lifecycle of projects and tasks, authenticates users, and maintains the system's state. The Control Plane does not execute agent code or process the raw data associated with a task. Its primary design optimization is for data consistency, reliability, and administrative control, ensuring that management functions remain available even under high load or during failures in the workload plane.4 By centralizing this logic, it provides a single point of management and simplifies system-wide policy enforcement.2  
* **The Execution Plane (Data Plane):** This is the "muscle" of the system, where the actual data movement and agent execution occur.1 This plane is composed of a fleet of stateless workers that provision and manage ephemeral, isolated environments—Docker containers for the MVP—to run the agent workloads. The Execution Plane is intentionally designed to be less complex than the Control Plane, with fewer moving parts, to minimize the statistical likelihood of failure.5 Its primary design optimization is for availability, high-throughput execution, and, most importantly, isolation. It receives instructions from the Control Plane but operates independently, ensuring that its workloads are sandboxed.

The rationale for this strict separation directly addresses the core security challenge: running potentially risky, autonomous code with elevated permissions safely. A compromise or catastrophic failure within an agent container in the Execution Plane is contained within that sandbox. Because the planes are decoupled, such an event cannot directly impact the core management logic, databases, or APIs of the Control Plane. This architectural boundary prevents a single agent failure from escalating into a system-wide outage or security breach, enabling a zero-trust security model where the Control Plane never implicitly trusts the Execution Plane.1

### **1.2 High-Level Architectural Diagram**

The following diagrams illustrate the dynamic and static relationships between the components of the Aetherium system. The sequence diagram shows the end-to-end flow of a task from initiation to the feedback loop, while the component diagram provides a static view of the system's structure.

Code snippet

graph TD  
    subgraph Control Plane  
        A\[API Gateway\] \--\>|routes| B(Task Orchestrator);  
        B \--\>|pushes task| C;  
        B \--\>|read/write state| D;  
        F \--\>|receives webhook| A;  
    end

    subgraph Execution Plane  
        E \--\>|pulls task| C;  
        E \--\>|manages| G{Docker Daemon};  
        G \--\>|runs| H\[Agent Container\];  
        H \--\>|streams logs| I\[Centralized Logging (Loki)\];  
        E \--\>|streams logs| I;  
        B \--\>|streams logs| I;  
        A \--\>|streams logs| I;  
    end

    subgraph External Services  
        J\[User\] \--\>|API calls| A;  
        K\[GitHub\] \<--\>|API/Webhooks| F(GitHub Integration Service);  
        F \--\>|creates PR| K;  
        J \<--\>|views PR, comments| K;  
        L\[Log Viewer (Grafana)\] \--\>|queries| I;  
    end

    E \--\>|sends output| F;

Code snippet

sequenceDiagram  
    participant User  
    participant API Gateway  
    participant Task Orchestrator  
    participant Task Queue  
    participant Agent Worker  
    participant Agent Container  
    participant GitHub Service  
    participant GitHub

    User-\>\>+API Gateway: POST /tasks (start agent)  
    API Gateway-\>\>+Task Orchestrator: Enqueue Task  
    Task Orchestrator-\>\>+Task Queue: Push task details  
    Task Queue--\>\>-Agent Worker: Pop task  
    Agent Worker-\>\>+Agent Container: docker run...  
    Agent Container--\>\>-Agent Worker: Execution & Output  
    Agent Worker-\>\>+GitHub Service: Send final output  
    GitHub Service-\>\>+GitHub: Create Branch, Commit, Open PR  
    GitHub--\>\>-User: Pull Request Created  
    User-\>\>+GitHub: Add Comment to PR  
    GitHub-\>\>+GitHub Service: Webhook Event  
    GitHub Service-\>\>+API Gateway: POST /webhooks/github  
    API Gateway-\>\>+Task Orchestrator: Enqueue follow-up task  
    Task Orchestrator--\>\>-Task Queue: Push new task with comment context

### **1.3 Component Interaction and Workflow**

The system operates through three primary, interconnected workflows that facilitate the entire lifecycle of an AI-driven task.

1. **Task Initiation and Orchestration:** A user initiates a task by sending a POST request to the /tasks endpoint of the API Gateway. This request contains the project context, the type of agent to use (e.g., "gemini-cli"), and the initial prompt. The API Gateway authenticates the request and forwards it to the Task Orchestrator. The Orchestrator creates a new task record in the State Database with a status of PENDING, assigns it a unique ID, and serializes a job message containing this ID and other necessary context. This message is then published to the Task Queue, effectively delegating the execution responsibility.  
2. **Agent Execution and Output Delivery:** An available Agent Worker, a stateless process in the Execution Plane, is constantly polling the Task Queue. It receives the job message, updates the task's status in the database to RUNNING, and begins the execution process. Using the Docker SDK for Go, the worker pulls the specified agent image (e.g., gemini-cli:latest) and runs it as a new, sandboxed container. The agent's output is captured from a mounted volume. Upon the container's completion, the worker updates the task status to COMPLETED and sends the generated artifacts (code, files, etc.) to the GitHub Integration Service. This service then uses the GitHub API to perform the necessary Git operations: creating a new branch, committing the files, and opening a pull request against the project's base branch. The URL of the newly created PR is saved back to the task record in the database.  
3. **The GitHub-Based Feedback Loop:** The user is notified of the new PR. They can review the agent's work directly within the familiar GitHub UI. To provide feedback or request further changes, the user simply adds a comment to the PR conversation, perhaps prefixed with a command like /aetherium run. This action triggers a configured GitHub webhook, which sends a payload to the GitHub Integration Service. The service validates the webhook's signature for security, parses the comment content, and makes an authenticated call back to the Aetherium API Gateway. This call initiates a new, linked task, using the original task as context and the new comment as the prompt. This cycle of PR \-\> comment \-\> new task \-\> updated PR forms the continuous, iterative "conversation" that allows for refining the agent's work.

This architectural approach, separating the "what" (the task definition in the Control Plane) from the "how" (the execution in the Execution Plane), is more than just a security measure. It directly implements the principles of a hierarchical multi-agent system.6 The Task Orchestrator functions as a high-level "supervisor" or "leader" agent, responsible for task decomposition (in this case, creating discrete, executable jobs) and delegation.6 The Agent Workers and their corresponding containers act as specialized "worker" agents, each equipped with the specific tools (the Claude or Gemini CLI) to execute a given task within their domain.6 This model provides a clear and scalable framework. Introducing a new agent type, such as one for image generation, would simply involve creating a new containerized worker agent and registering it with the Orchestrator. System-wide logic, such as task prioritization or cost management, can be enhanced centrally within the Orchestrator without altering the worker agents. This structure mirrors real-world organizations and is a proven pattern for managing complexity in AI systems.6

## **Section 2: The Control Plane: System Orchestration and Management**

The Control Plane is the authoritative core of Aetherium, housing the components responsible for managing the system's logic, state, and API. It is designed for consistency and reliability, serving as the single source of truth for all projects, tasks, and their associated metadata. The services in this plane will be implemented in Go, leveraging its performance and strong concurrency primitives.

### **2.1 The API Gateway: The System's Front Door**

All external interactions with Aetherium are managed through a single, well-defined API Gateway. This component serves as the system's front door, providing a unified interface for users and external services like the GitHub webhook handler. Its responsibilities include request authentication, authorization, rate limiting, and routing to the appropriate backend services. The API will be designed as a RESTful interface, adhering to established industry best practices to ensure it is intuitive, predictable, and easy for clients to consume.10 For implementation in Go, a high-performance and minimalist framework such as Gin or Echo is recommended.12

The design will be guided by the following principles:

* **Resource-Oriented Naming:** Endpoints will be structured around nouns that represent the system's entities, such as projects, tasks, and logs. The action to be performed is implied by the HTTP method, not included in the URI path (e.g., POST /tasks, not /createTask).10  
* **Plural Nouns for Collections:** URIs that reference a collection of resources will use plural nouns, creating a clear and hierarchical structure (e.g., /projects for the collection of all projects, and /projects/{project\_id}/tasks for the collection of tasks within a specific project).10  
* **Standard HTTP Methods:** The API will use the standard HTTP verbs consistently for CRUD (Create, Read, Update, Delete) operations. GET will be used for retrieval, POST for creation, and DELETE for termination or removal.11  
* **JSON for Data Interchange:** All request and response bodies will use the application/json media type. This is the de-facto standard for modern APIs and is universally supported by client-side and server-side technologies.11  
* **Graceful Error Handling:** The API will return standard HTTP status codes to indicate the outcome of a request, especially for errors. This allows clients to programmatically handle different failure scenarios. Common codes will include 400 Bad Request for client-side validation errors, 401 Unauthorized for authentication failures, 404 Not Found for missing resources, and 500 Internal Server Error for unexpected server-side issues.11

A well-defined API contract is essential for any distributed system, as it enables parallel development and serves as the primary documentation. The following table specifies the core endpoints for the Aetherium API.

**Table: Core API Endpoints**

| Endpoint | Method | Description | Payload Example | Success Response |
| :---- | :---- | :---- | :---- | :---- |
| /projects | POST | Create a new project workspace. | {"name": "claude-refactor", "repo\_url": "..."} | 201 Created, Project object |
| /projects/{id} | GET | Get details of a specific project. | N/A | 200 OK, Project object |
| /tasks | POST | Start a new agent task. | {"project\_id": "...", "agent\_type": "gemini", "prompt": "..."} | 202 Accepted, Task object with status: PENDING |
| /tasks/{id} | GET | Get the current status of a task. | N/A | 200 OK, Task object with status, logs URL, PR URL |
| /tasks/{id} | DELETE | Terminate a running agent task. | N/A | 202 Accepted |
| /tasks/{id}/logs | GET | Stream logs from a running agent container. | N/A | 200 OK, Streaming log data |
| /webhooks/github | POST | Internal endpoint for receiving PR comment webhooks. | GitHub Webhook Payload | 200 OK |

### **2.2 The Task Orchestrator: The Brains of the Operation**

At the heart of the Control Plane lies the Task Orchestrator. This service embodies the core business logic of the system. It receives validated task requests from the API Gateway, persists them to the State Database, and delegates the actual execution by placing a corresponding job onto a message queue.

A critical design decision for the Orchestrator is the selection of the task queue system. For a system built in Go, several robust options exist:

* **Asynq:** A simple, reliable, and efficient distributed task queue library backed by Redis.15 It is designed for scalability and ease of use, providing features like scheduling, retries, a web UI for monitoring, and task prioritization through different queues.15 Its focus on simplicity and performance makes it an excellent choice for this architecture.  
* **Machinery:** A more flexible, broker-agnostic distributed task queue. It supports RabbitMQ, Redis, and AWS SQS as backends, offering greater interoperability.17 This flexibility is valuable for complex workflows or integration with existing messaging systems.17

For the Aetherium system, where tasks can be long-running and their successful completion is critical, reliability and observability are paramount. The recommended choice is **Asynq with Redis as the message broker**. This combination provides a modern, simple-to-use library with a strong feature set for production environments, including a monitoring UI and automatic retries.16 This architecture ensures that once a task is accepted by the Orchestrator, it will not be lost due to a transient failure or crash of a worker node, a crucial feature for a production-grade system.

### **2.3 State and History Persistence**

To maintain a durable record of all system activities, a relational database, such as PostgreSQL, will serve as the primary state store. This database will house information about projects, the status of individual tasks, and the history of agent interactions.

The database schema will be designed to support the system's core workflows:

* **projects table:** This table will store information about the high-level workspaces.  
  * Columns: id (primary key), name (text), github\_repo\_url (text), created\_at (timestamp).  
* **tasks table:** This is the central table for tracking the lifecycle of each agent run.  
  * Columns: id (primary key), project\_id (foreign key to projects), parent\_task\_id (self-referencing foreign key to enable threading of conversations from the GitHub feedback loop), status (enum: PENDING, RUNNING, COMPLETED, FAILED, TERMINATED), agent\_type (text), prompt (text), container\_id (text, nullable), pr\_url (text, nullable), created\_at (timestamp), completed\_at (timestamp, nullable).  
* **agent\_history table:** This table is crucial for enabling more advanced, context-aware agent behaviors in the future. It stores a persistent record of the conversation flow for each task thread, as maintaining conversation history is a key component of sophisticated agent architectures.6  
  * Columns: id (primary key), task\_id (foreign key to tasks), interaction\_turn (integer), content (text, storing either the user's prompt or the agent's output).

This structured, relational data model provides a solid foundation for querying system status, generating audit trails, and building future features that rely on historical context.

## **Section 3: The Execution Plane: Secure and Isolated Agent Runtimes**

The Execution Plane is where the core work of the AI agents is performed. Its design is driven by an uncompromising focus on security, isolation, and resource management to ensure that autonomous agent execution never poses a threat to the host system or other workloads.

### **3.1 The Agent Worker Fleet**

The Execution Plane is populated by a fleet of one or more stateless Agent Worker services. These workers, written in Go, are the bridge between the Control Plane's commands and the actual containerized execution. Their sole responsibility is to connect to the Task Queue, consume job messages, and manage the complete lifecycle of agent containers using the official Docker Engine SDK for Go.18

The implementation of the container lifecycle management within the worker will be as follows:

* **Start:** Upon receiving a task, the worker will first create the container using cli.ContainerCreate(...). This call configures the container with all the security and resource constraints detailed in Section 3.3. The worker will then start the container with cli.ContainerStart(...).19 The returned container ID is immediately stored in the State Database, updating the task's status to  
  RUNNING.  
* **Monitor Logs:** For real-time observability, the worker will attach to the container's log stream using cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true}). The resulting stream will be read, and each line will be forwarded to the centralized logging system (Loki), enriched with metadata like the task\_id and container\_id.19  
* **Stop/Terminate:** The system must support forceful termination of tasks. When a user issues a DELETE /tasks/{id} request, the Orchestrator updates the task's desired state to TERMINATED. The Agent Worker, periodically checking the status of its active tasks, will detect this change. It will then use cli.ContainerStop(ctx, containerID, container.StopOptions{}) or, if necessary, cli.ContainerKill(ctx, containerID, "SIGKILL") to forcefully terminate the process.

### **3.2 A "Batteries-Included" Agent Execution Environment**

To provide the AI agents with a versatile and powerful workspace, the system will depart from the minimal-image philosophy and instead adopt a "batteries-included" base image. This approach equips the agent with a comprehensive Linux environment, pre-loaded with a suite of common development tools, programming languages, and even the Docker daemon itself. This turns the container from a single-purpose executable into a persistent, stateful workspace where the agent can clone repositories, install dependencies, and execute complex, multi-step tasks.

The base image will be built on a standard Linux distribution (e.g., Ubuntu) and will include:

* **Core Utilities:** Git, curl, wget, and other essential command-line tools.  
* **Language Runtimes:** Pre-installed environments for popular languages such as Go, Python, and Node.js, along with their respective package managers.  
* **Containerization Tools:** A fully functional Docker daemon and CLI, enabling Docker-in-Docker (DinD) operations. This allows an agent to build and run its own containers, a powerful capability for tasks like building application artifacts or running integration tests.

A template Dockerfile for this workspace environment would look like this:

Dockerfile

\# Start from a full-featured base OS  
FROM ubuntu:22.04

\# Install essential packages, language runtimes, and Docker  
RUN apt-get update && apt-get install \-y \\  
    git \\  
    curl \\  
    python3-pip \\  
    nodejs \\  
    docker.io \\  
    && rm \-rf /var/lib/apt/lists/\*

\# Install Go  
RUN apt-get update && apt-get install \-y golang-go

\# Create a dedicated, non-root user for agent execution  
RUN groupadd \-r agentuser && useradd \--no-log-init \-r \-g agentuser agentuser  
USER agentuser  
WORKDIR /home/agentuser/workspace

\# Entrypoint script to start Docker daemon and keep container alive  
COPY entrypoint.sh /usr/local/bin/  
ENTRYPOINT \["entrypoint.sh"\]

It is critical to acknowledge that enabling Docker-in-Docker requires running the outer container in \--privileged mode. This significantly weakens the isolation boundary between the container and the host kernel, effectively granting the container root-level access to the host. This is a direct trade-off between functionality and security and must be carefully managed. The hardening measures in Section 3.3 are still crucial but are less effective against a potential container escape from a privileged container.21

#### **3.2.1 Declarative Workspace Provisioning with Nix Flakes**

While the base image provides a general-purpose toolset, each specific task will require a unique set of dependencies for the target repository. Managing these dependencies imperatively (e.g., via pip install or npm install) can lead to conflicts and non-reproducible environments. To solve this, Aetherium will leverage **Nix Flakes** to declaratively and reproducibly provision the exact environment needed for each task.58

Nix Flakes is an experimental but powerful feature of the Nix package manager that allows developers to define a project's complete dependency graph in a flake.nix file. This file, when combined with a flake.lock file, pins every dependency to a specific version, guaranteeing a bit-for-bit identical environment every time it's created.59

The workflow for setting up a repository-specific workspace will be as follows:

1. The Agent Worker starts the "batteries-included" container and clones the target repository into its workspace.  
2. The worker inspects the repository for a flake.nix file.  
3. If a flake.nix is found, the worker executes the command nix develop inside the container. This command reads the flake definition and creates an isolated sub-shell containing all the specified tools, libraries, and environment variables, without polluting the base container environment.58  
4. The AI agent then executes all subsequent commands (e.g., running the Gemini CLI, building code, running tests) within this perfectly configured and reproducible Nix shell.

This approach provides the best of both worlds: a versatile base image with common tools, and a highly specific, declarative, and reproducible environment for each individual task. While other tools like Cloud Native Buildpacks and Railpacks also manage dependencies, they are primarily designed to produce a final OCI image from source code.62 Nix Flakes, in contrast, are explicitly designed to create reproducible

*development environments*, which aligns perfectly with the agent's need for a functional workspace.

### **3.3 Hardened Container Sandboxing: A Defense-in-Depth Approach**

This is the most critical aspect of the Execution Plane's design. A multi-layered, defense-in-depth security posture will be applied to every agent container, based on the OWASP Docker Security Cheat Sheet and other industry best practices.20

The user's request for agents with "elevated permissions" within a "safe environment" presents a fundamental conflict. Powerful autonomous agents may need to perform actions that require privileges, such as file system writes, network access, or even installing new packages to fulfill a request. However, standard container hardening practices are built on the principle of least privilege: removing all permissions and capabilities that are not strictly necessary.20 Granting more permissions to a process inside a container inherently increases the attack surface of the shared host kernel. A vulnerability in the kernel could be exploited by a compromised agent to escape the container and gain control of the host.

This inherent tension makes it clear that while Docker is suitable for an MVP, it is not the ideal long-term solution for running untrusted, powerful agents. The ultimate solution, hardware-level isolation via microVMs, is discussed in Section 6\. For the initial Docker-based implementation, the system must apply the strictest possible lockdown that still allows the agent to perform its core function, while explicitly acknowledging and mitigating the residual risk of a shared kernel.

The following table provides a concrete checklist of security controls that the Agent Worker must enforce when launching any container. It translates abstract principles into specific Go Docker SDK configuration fields, making it directly actionable for developers.

**Table: Container Security Configuration for Agent Worker (Go SDK)**

| Security Control | Go Docker SDK Field (container.HostConfig) | Rationale & Snippet Reference |
| :---- | :---- | :---- |
| **Run as Non-Root User** | container.Config.User \= "1000" | Prevents processes inside the container from running as root, mitigating a large class of privilege escalation attacks. 21 |
| **Prevent New Privileges** | HostConfig.SecurityOpt \=string{"no-new-privileges:true"} | Disables setuid and setgid binaries from granting additional privileges to a process, a common escalation vector. 21 |
| **Drop All Capabilities** | HostConfig.CapDrop \=string{"ALL"} | Enforces the principle of least privilege by removing all default Linux kernel capabilities. Specific, minimal capabilities can be added back via CapAdd. 20 |
| **Read-Only Root Filesystem** | HostConfig.ReadonlyRootfs \= true | Prevents the agent from modifying its own runtime environment or installing unauthorized software, neutralizing many common attack techniques. 21 |
| **Resource Limits (CPU/Mem)** | HostConfig.Resources.Memory \= 512 \* 1024 \* 1024 HostConfig.Resources.CPUShares \= 512 | Protects the host from Denial of Service (DoS) attacks by preventing a runaway or malicious agent from consuming all available CPU and memory. 21 |
| **Network Isolation** | HostConfig.NetworkMode \= "\<custom\_bridge\>" | Attaches containers to a custom, non-default bridge network. This allows for fine-grained firewall rules and prevents accidental communication with other containers on the host. 22 |
| **Disable Privileged Mode** | HostConfig.Privileged \= false | **This is a non-negotiable, critical security setting.** Setting this to true would grant the container full root access to the host, completely bypassing all other isolation mechanisms. 21 |

### **3.4 Secure Credential Injection**

Agents will inevitably require sensitive credentials, such as API keys for the GitHub, Claude, and Gemini services. These secrets must be managed securely throughout their lifecycle. Common but insecure practices, like hardcoding secrets into Docker images or passing them as environment variables, will be strictly avoided. Environment variables are easily inspectable by any process within the container and are often inadvertently leaked into logs.27

The correct approach is to use **Docker Secrets** for runtime credential management. This feature allows secrets to be securely transmitted to and mounted within only the containers that need them, without persisting them in the image or exposing them broadly.27

The implementation workflow will be as follows:

1. The Control Plane will securely retrieve the necessary API token for a given task from a central vault (e.g., HashiCorp Vault, AWS Secrets Manager).  
2. This token will be passed to the Agent Worker as part of the job payload over a secure channel.  
3. The worker will create a temporary secret specific to that single task using the Go Docker SDK.  
4. When creating the agent container via cli.ContainerCreate(...), the worker will use the HostConfig.Secrets field to mount the secret into the container's in-memory filesystem, typically at /run/secrets/\<secret\_name\>.28  
5. The agent's code will be designed to read the API key from this specific file path.  
6. Once the container terminates, Docker ensures the in-memory secret is automatically destroyed. This just-in-time, ephemeral approach dramatically reduces the exposure window for sensitive credentials.30

## **Section 4: The GitHub Feedback Loop: From Output to Interaction**

A novel feature of the Aetherium platform is its use of GitHub as the primary user interface for agent interaction. This design choice transforms a standard developer workflow into an event-driven control surface, making the process of iterating with an AI agent both intuitive and fully auditable.

### **4.1 The GitHub Integration Service**

To centralize and isolate all interactions with the GitHub API, a dedicated microservice, the GitHub Integration Service, will be implemented in Go. This service acts as a facade, abstracting away the complexities of Git operations and webhook handling from the rest of the system.

Its core responsibilities are:

1. **PR Creation:** Receiving the final output (code, documentation, etc.) and context (repository URL, base branch) from a completed Agent Worker task.  
2. **Authentication:** Securely authenticating with the GitHub API using a GitHub App token or a fine-grained Personal Access Token (PAT), stored in a secure vault.  
3. **Git Workflow Execution:** Programmatically performing the sequence of Git operations required to deliver the agent's work: creating a new branch, committing the generated files, and opening a pull request.  
4. **Webhook Ingestion:** Providing a secure endpoint to receive, validate, and process incoming webhooks from GitHub, specifically for comments made on pull requests, to enable the feedback loop.

### **4.2 Implementation: Creating a Pull Request with Go**

This workflow will be implemented in Go using the **go-github** library, which provides a clean, idiomatic client for the GitHub REST API, simplifying development.32

The step-by-step process within the GitHub Integration Service to create a PR will be:

1. **Authentication:** Instantiate the go-github client with an http.Client configured for OAuth2 authentication.  
   Go  
   import (  
       "context"  
       "github.com/google/go-github/v63/github"  
       "golang.org/x/oauth2"  
   )

   ctx := context.Background()  
   ts := oauth2.StaticTokenSource(  
       \&oauth2.Token{AccessToken: "YOUR\_GITHUB\_TOKEN"},  
   )  
   tc := oauth2.NewClient(ctx, ts)  
   client := github.NewClient(tc)

2. **Get Base Branch Reference:** Obtain the repository's default branch and the SHA of its latest commit. This SHA will be the starting point for the new agent branch.  
   Go  
   repo, \_, err := client.Repositories.Get(ctx, "owner", "repo")  
   //... error handling...  
   baseBranch, \_, err := client.Repositories.GetBranch(ctx, "owner", "repo", \*repo.DefaultBranch, 3)  
   baseCommitSHA := \*baseBranch.Commit.SHA

3. **Create New Branch:** Create a new Git reference (a branch) pointing to the base commit SHA. The branch name should be unique and descriptive.  
   Go  
   newBranchName := "aetherium-task-" \+ taskID  
   ref := "refs/heads/" \+ newBranchName  
   \_, \_, err \= client.Git.CreateRef(ctx, "owner", "repo", \&github.Reference{  
       Ref: \&ref,  
       Object: \&github.GitObject{  
           SHA: \&baseCommitSHA,  
       },  
   })

4. **Commit File(s):** For multiple files, the GitHub API requires a multi-step process: create a blob for each file's content, create a tree object that references these blobs, and finally create a commit that points to this new tree.  
   Go  
   // This is a simplified example for one file. A real implementation would loop over files.  
   entry := github.TreeEntry{  
       Path:    github.String("path/to/new-file.txt"),  
       Type:    github.String("blob"),  
       Content: github.String("File content here"),  
       Mode:    github.String("100644"),  
   }  
   tree, \_, err := client.Git.CreateTree(ctx, "owner", "repo", baseCommitSHA,github.TreeEntry{entry})

   commit, \_, err := client.Git.CreateCommit(ctx, "owner", "repo", "feat: Add new file", tree,\*github.Commit{{SHA: \&baseCommitSHA}})

   // Update the new branch to point to the new commit  
   refObj, \_, err := client.Git.UpdateRef(ctx, "owner", "repo", ref, \*commit.SHA, false)

5. **Create Pull Request:** Finally, create the pull request, targeting the repository's default branch from the newly created agent branch.  
   Go  
   prBody := "This PR was generated by Aetherium task \`" \+ taskID \+ "\`."  
   newPR := \&github.NewPullRequest{  
       Title: github.String("Aetherium Task: " \+ taskID),  
       Head:  github.String(newBranchName),  
       Base:  repo.DefaultBranch,  
       Body:  github.String(prBody),  
   }  
   pr, \_, err := client.PullRequests.Create(ctx, "owner", "repo", newPR)

### **4.3 Closing the Loop: Ingesting Feedback via Webhooks**

The interactive loop is closed by configuring a GitHub Webhook on the target repositories. This webhook transforms user actions within the GitHub UI into events that trigger new tasks in Aetherium.34

The webhook will be configured as follows:

* **Payload URL:** This will point to the secure /webhooks/github endpoint exposed by the GitHub Integration Service.  
* **Content Type:** application/json.  
* **Secret:** A strong, randomly generated secret must be configured on both GitHub and the Integration Service. This secret is used to sign the webhook payload, allowing the service to verify that incoming requests are legitimate and have not been tampered with.34  
* **Events:** The webhook should subscribe specifically to the issue\_comment event with an action of created. While other events like pull\_request\_review\_comment exist, issue\_comment is more suitable for general, high-level instructions posted in the main PR conversation timeline, rather than comments on specific lines of code.34

The logic within the webhook handler of the Integration Service will be:

1. Upon receiving a POST request, immediately compute the HMAC SHA256 signature of the request body using the shared secret and compare it to the X-Hub-Signature-256 header from GitHub. If they do not match, the request is discarded.  
2. If the signature is valid, parse the JSON payload to extract the comment body, the PR number (which is an issue number in the context of the API), and the repository details.  
3. To prevent every comment from triggering a new agent run, the handler will look for a specific trigger phrase, such as /aetherium run, at the beginning of the comment.  
4. If the trigger phrase is present, the handler will extract the rest of the comment as the new prompt. It will also parse the original PR body to find the parent\_task\_id.  
5. Finally, it will make an authenticated API call to the Control Plane's POST /tasks endpoint, creating a new task linked to the original via the parent\_task\_id and providing the new prompt from the user's comment. This action enqueues a new job and restarts the execution cycle.

## **Section 5: System Observability and Monitoring**

For a system running autonomous agents, robust observability is not a luxury but a core operational requirement. Operators must have full visibility into the system's state, the ability to debug failures, and real-time access to the activities of any running agent.

### **5.1 A Modern, Lightweight Logging Pipeline with the Grafana Loki Stack**

To manage logs from the various distributed components of Aetherium, a centralized logging pipeline will be established using the Grafana Loki stack. This stack, consisting of Loki, Promtail, and Grafana, provides a powerful, lightweight, and scalable platform for collecting, searching, and visualizing log data, making it an excellent modern alternative to the more resource-intensive ELK stack.38

The logging architecture is designed for cost-effectiveness and ease of operation, especially in containerized environments.38 Its core principle is to index only the metadata (labels) associated with log streams, rather than the full text of the logs themselves.39 The log data is then compressed into chunks and stored in a low-cost object store like AWS S3.42 This approach dramatically reduces storage costs and operational complexity.39

The stack consists of the following components:

* **Promtail:** A lightweight log collection agent specifically designed for Loki.38 Promtail will be deployed to each host in the Execution Plane. It discovers running containers, scrapes their logs, attaches relevant metadata as labels (e.g.,  
  container\_name, job), and pushes the log streams to the central Loki instance.44 While other agents like Fluent Bit can also be used, Promtail is recommended for its simplicity and tight integration with the Loki ecosystem.45  
* **Loki:** The horizontally-scalable, multi-tenant log aggregation system.44 Loki receives log streams from Promtail, indexes the labels, and stores the compressed log chunks in an object store.42 Its microservices-based architecture allows it to scale from small deployments to handling massive log volumes.42  
* **Grafana:** A powerful, open-source visualization tool that provides the user interface for the logging system.38 Loki is a native data source in Grafana, allowing operators to seamlessly query, visualize, and correlate logs with other metrics (like those from Prometheus) in a single dashboard.39

This centralized approach is invaluable for debugging. If a task fails, an operator can search for its unique task\_id label in Grafana and immediately see an interleaved view of logs from the API Gateway, the Task Orchestrator, the specific Agent Worker, and the ephemeral Agent Container itself, providing a complete, end-to-end trace of the failed operation.

### **5.2 Real-time Log Streaming and Status Checks**

While the Loki stack is ideal for historical analysis and debugging, operators also need immediate, real-time access to the status and output of currently running tasks. This will be provided through dedicated API endpoints.

* **Status Endpoint (GET /tasks/{id}):** This endpoint provides a snapshot of a task's current state. When called, the API Gateway will query the State Database for the specified task\_id. It will return a JSON object containing the task's current status (e.g., PENDING, RUNNING, COMPLETED), timestamps, and, once available, the URL to the final GitHub pull request. This allows for programmatic polling of a task's progress.  
* **Log Streaming Endpoint (GET /tasks/{id}/logs):** This endpoint provides a direct, low-latency way for a user to "tail" the logs of a specific agent run as it happens. Rather than querying the potentially delayed data in Loki, this request will be handled differently. The API Gateway will first query the State Database to retrieve the container\_id associated with the task\_id. It will then use the Docker SDK for Go to connect directly to the Docker daemon managing that container and stream its log output back to the client via a chunked HTTP response. This is achieved by calling cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, Follow: true}) on the appropriate container object.19 This provides the immediate feedback necessary for monitoring active agent behavior.

## **Section 6: The Path Forward: Evolving to a MicroVM Architecture**

The initial MVP of Aetherium, built on hardened Docker containers, provides a strong security baseline. However, the fundamental architecture of containers involves a shared kernel between all workloads on a host. This shared resource represents a significant attack surface; a single, sufficiently severe kernel exploit could allow a malicious agent to escape its sandbox and compromise the entire host system. To address this inherent risk and fulfill the long-term vision of running truly untrusted, powerful AI agents with confidence, the system is designed to evolve towards a microVM-based architecture.

### **6.1 A Comparative Analysis: Docker vs. Kata Containers vs. Firecracker**

The evolution from containers to microVMs involves choosing the right virtualization technology. The leading candidates each offer a different balance of security, performance, and ecosystem compatibility.51

* **Docker (OS-level Virtualization):**  
  * **Isolation:** Provides isolation at the operating system level using Linux namespaces and cgroups. All containers on a host share the same Linux kernel.  
  * **Security:** Security relies on limiting the container's privileges and capabilities (seccomp, AppArmor, dropping capabilities). The shared kernel is the primary point of failure.  
  * **Performance:** Offers near-native performance with very low startup overhead, as no new kernel needs to be booted.  
* **Firecracker (Hardware Virtualization):**  
  * **Isolation:** Provides true hardware-level isolation using a KVM-based Virtual Machine Monitor (VMM). Each Firecracker microVM runs its own guest kernel, completely separate from the host kernel and other microVMs.  
  * **Security:** Offers the strongest level of isolation, effectively eliminating the shared kernel attack surface. It is purpose-built with a minimalist design to run serverless functions and containers securely.51  
  * **Performance:** Engineered for extremely fast boot times (under 150ms) and a very low memory footprint, making it far more lightweight than traditional VMs. However, there is still a performance overhead compared to standard Docker containers.52  
* **Kata Containers (Hardware Virtualization with OCI Compliance):**  
  * **Isolation:** Also uses hardware virtualization to run containers inside lightweight VMs. Crucially, Kata Containers is an OCI (Open Container Initiative) compliant runtime. This means it can act as a drop-in replacement for standard runtimes like runc in a containerd or CRI-O environment.  
  * **Security:** Provides the same strong, VM-based isolation boundary. A key feature is its ability to use different underlying hypervisors, including the highly optimized Firecracker VMM.53  
  * **Performance:** Performance is similar to other lightweight VM solutions. Its main advantage is not raw speed but its seamless integration into the existing container ecosystem, abstracting away the complexity of managing VMs.53

### **6.2 A Phased Migration Strategy**

The recommended path forward is to adopt **Kata Containers using the Firecracker hypervisor**. This strategy provides the best of both worlds: the unparalleled security and performance of Firecracker's microVMs, combined with the ecosystem compatibility and operational simplicity of an OCI-compliant runtime like Kata.53 This avoids the significant engineering effort required to build a custom integration with the Firecracker API directly.56

The migration from the Docker-based Execution Plane to a Kata-based one can be achieved in a phased manner, thanks to the decoupled architecture:

1. **Prepare the Execution Plane Hosts:** The host machines that run the Agent Workers will need to be provisioned with the necessary components: containerd as the container runtime, and the Kata Containers packages, including the Firecracker VMM binary.  
2. **Update the Agent Worker Service:** The code within the Agent Worker will be modified. Instead of using the Go Docker SDK, it will use a client library for containerd.  
3. **Specify the RuntimeClass:** The most significant change in the worker's logic will be specifying the RuntimeClass when creating a workload. A RuntimeClass is a Kubernetes and containerd feature that allows selecting which OCI runtime to use. The worker will specify the kata-fc (Kata-Firecracker) runtime class. This instruction tells containerd to delegate the creation of the pod to Kata Containers, which will then launch a Firecracker microVM to house the agent container, instead of using the default runc runtime.53  
4. **Leverage Existing Agent Images:** A major benefit of this approach is that the Docker images built for the agents in Section 3.2 remain **fully compatible and require no changes**. Because Kata is OCI-compliant, it can run any standard container image.55 This decouples the agent development workflow from the underlying execution infrastructure.  
5. **No Impact on the Control Plane:** The Control Plane, GitHub Integration Service, and logging pipeline remain entirely unchanged. The Task Orchestrator continues to place jobs on the queue, unaware of whether the Execution Plane is running those jobs in Docker containers or Firecracker microVMs. This demonstrates the profound value of the initial architectural decision to strictly separate the planes.

This phased migration allows Aetherium to start with a well-understood and rapidly deployable Docker-based MVP and evolve gracefully to a state-of-the-art, security-hardened production system without requiring a fundamental rewrite.

## **Conclusion and Strategic Recommendations**

The Aetherium architecture presented in this report provides a comprehensive and robust blueprint for building a secure, manageable, and developer-friendly platform for autonomous AI agents. By grounding the design in the proven paradigm of a separated Control and Execution Plane, it addresses the primary challenge of safely executing untrusted code. The integration of a GitHub-based workflow creates an intuitive and auditable feedback loop, while the commitment to centralized observability ensures operational transparency.

The design is not merely a static snapshot but an evolutionary path. It begins with a practical, Docker-based MVP that can be implemented rapidly, while clearly defining the necessary steps to transition to a more secure microVM-based runtime as the system matures. This ensures that the platform can grow in sophistication and security without accumulating prohibitive architectural debt.

For the team tasked with implementing Aetherium, the following strategic recommendations are paramount:

1. **Prioritize the Control/Execution Plane Separation:** This foundational pattern is the most critical architectural decision. It must be implemented from day one, as it underpins the entire system's security, scalability, and future evolution. Do not compromise on this separation for short-term expediency.  
2. **Implement All Docker Hardening Measures for the MVP:** The security controls detailed in Section 3.3 should be treated as a mandatory checklist, not a list of suggestions. They represent the minimum viable security posture for the initial Docker-based system and are essential for mitigating the risks of a shared-kernel architecture.  
3. **Build the GitHub Integration as a Decoupled Service:** Isolating all GitHub API interactions and webhook logic into a dedicated service will simplify the design of other components, improve modularity, and make the system easier to maintain and test.  
4. **Plan for the MicroVM Migration:** While the MVP will use Docker, all engineering and infrastructure decisions should be made with the eventual migration to Kata Containers in mind. This forward-looking perspective will prevent architectural choices that could become dead-ends, ensuring a smooth transition to a hardware-isolated security model in the future.

By adhering to these principles and the detailed design laid out in this document, it is possible to build a powerful platform that unlocks the potential of autonomous AI agents while upholding the highest standards of security and operational excellence.

#### **Works cited**

1. Control Plane vs. Data Plane: Key Differences Explained \- Estuary, accessed September 13, 2025, [https://estuary.dev/blog/control-plane-vs-data-plane/](https://estuary.dev/blog/control-plane-vs-data-plane/)  
2. Control Plane vs. Data Plane \- IBM, accessed September 13, 2025, [https://www.ibm.com/think/topics/control-plane-vs-data-plane](https://www.ibm.com/think/topics/control-plane-vs-data-plane)  
3. Chapter 2\. Control plane adjustments | Red Hat Ansible Automation Platform Performance Considerations for Operator Based Installations, accessed September 13, 2025, [https://docs.redhat.com/en/documentation/red\_hat\_ansible\_automation\_platform/2.3/html/red\_hat\_ansible\_automation\_platform\_performance\_considerations\_for\_operator\_based\_installations/assembly-control-plane-adjustments](https://docs.redhat.com/en/documentation/red_hat_ansible_automation_platform/2.3/html/red_hat_ansible_automation_platform_performance_considerations_for_operator_based_installations/assembly-control-plane-adjustments)  
4. Data and control planes for routing control \- Amazon Application Recovery Controller (ARC), accessed September 13, 2025, [https://docs.aws.amazon.com/r53recovery/latest/dg/data-and-control-planes.html](https://docs.aws.amazon.com/r53recovery/latest/dg/data-and-control-planes.html)  
5. Control planes and data planes \- AWS Fault Isolation Boundaries \- AWS Documentation, accessed September 13, 2025, [https://docs.aws.amazon.com/whitepapers/latest/aws-fault-isolation-boundaries/control-planes-and-data-planes.html](https://docs.aws.amazon.com/whitepapers/latest/aws-fault-isolation-boundaries/control-planes-and-data-planes.html)  
6. Designing Multi-Agent Intelligence \- Microsoft for Developers, accessed September 13, 2025, [https://developer.microsoft.com/blog/designing-multi-agent-intelligence](https://developer.microsoft.com/blog/designing-multi-agent-intelligence)  
7. Understanding AI agent types: A guide to categorizing complexity \- Red Hat, accessed September 13, 2025, [https://www.redhat.com/en/blog/understanding-ai-agent-types-simple-complex](https://www.redhat.com/en/blog/understanding-ai-agent-types-simple-complex)  
8. Hierarchical Multi-Agent Systems: Concepts and Operational Considerations \- Over Coffee, accessed September 13, 2025, [https://overcoffee.medium.com/hierarchical-multi-agent-systems-concepts-and-operational-considerations-e06fff0bea8c](https://overcoffee.medium.com/hierarchical-multi-agent-systems-concepts-and-operational-considerations-e06fff0bea8c)  
9. AI Agent Architectures: Patterns, Applications, and Implementation Guide \- DZone, accessed September 13, 2025, [https://dzone.com/articles/ai-agent-architectures-patterns-applications-guide](https://dzone.com/articles/ai-agent-architectures-patterns-applications-guide)  
10. Web API Design Best Practices \- Azure Architecture Center ..., accessed September 13, 2025, [https://learn.microsoft.com/en-us/azure/architecture/best-practices/api-design](https://learn.microsoft.com/en-us/azure/architecture/best-practices/api-design)  
11. Best practices for REST API design \- The Stack Overflow Blog, accessed September 13, 2025, [https://stackoverflow.blog/2020/03/02/best-practices-for-rest-api-design/](https://stackoverflow.blog/2020/03/02/best-practices-for-rest-api-design/)  
12. Top 18 Golang Web Frameworks to Use in 2025 \- Bacancy Technology, accessed September 14, 2025, [https://www.bacancytechnology.com/blog/golang-web-frameworks](https://www.bacancytechnology.com/blog/golang-web-frameworks)  
13. Go: The fastest web framework in 2025 | Tech Tonic \- Medium, accessed September 14, 2025, [https://medium.com/deno-the-complete-reference/go-the-fastest-web-framework-in-2025-dfa2ddfd09e9](https://medium.com/deno-the-complete-reference/go-the-fastest-web-framework-in-2025-dfa2ddfd09e9)  
14. The 8 best Go web frameworks for 2025: Updated list \- LogRocket Blog, accessed September 14, 2025, [https://blog.logrocket.com/top-go-frameworks-2025/](https://blog.logrocket.com/top-go-frameworks-2025/)  
15. golang: libs for managing work queues – pepa.holla.cz, accessed September 14, 2025, [https://pepa.holla.cz/2022/11/15/golang-libs-for-managing-work-queues/](https://pepa.holla.cz/2022/11/15/golang-libs-for-managing-work-queues/)  
16. Supercharging Go with Asynq: Scalable Background Jobs Made Easy \- DEV Community, accessed September 14, 2025, [https://dev.to/lovestaco/supercharging-go-with-asynq-scalable-background-jobs-made-easy-32do](https://dev.to/lovestaco/supercharging-go-with-asynq-scalable-background-jobs-made-easy-32do)  
17. Task Queues in Go: Asynq vs Machinery vs Work: Powering Background Jobs in High-Throughput Systems | by Geison \- Medium, accessed September 14, 2025, [https://medium.com/@geisonfgfg/task-queues-in-go-asynq-vs-machinery-vs-work-powering-background-jobs-in-high-throughput-systems-45066a207aa7](https://medium.com/@geisonfgfg/task-queues-in-go-asynq-vs-machinery-vs-work-powering-background-jobs-in-high-throughput-systems-45066a207aa7)  
18. Use containers for Go development \- Docker Docs, accessed September 14, 2025, [https://docs.docker.com/guides/golang/develop/](https://docs.docker.com/guides/golang/develop/)  
19. Examples using the Docker Engine SDKs and Docker API, accessed September 14, 2025, [https://docs.docker.com/reference/api/engine/sdk/examples/](https://docs.docker.com/reference/api/engine/sdk/examples/)  
20. 21 Docker Security Best Practices: Daemon, Image, Containers \- Spacelift, accessed September 13, 2025, [https://spacelift.io/blog/docker-security](https://spacelift.io/blog/docker-security)  
21. Docker Security \- OWASP Cheat Sheet Series, accessed September 13, 2025, [https://cheatsheetseries.owasp.org/cheatsheets/Docker\_Security\_Cheat\_Sheet.html](https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html)  
22. A no-BS Docker security checklist for the vulnerability-minded developer, accessed September 13, 2025, [https://www.aikido.dev/blog/a-no-bs-docker-security-checklist-for-the-vulnerability-minded-developer](https://www.aikido.dev/blog/a-no-bs-docker-security-checklist-for-the-vulnerability-minded-developer)  
23. Container Security Checklist: Importance & Mistakes \- SentinelOne, accessed September 13, 2025, [https://www.sentinelone.com/cybersecurity-101/cloud-security/container-security-checklist/](https://www.sentinelone.com/cybersecurity-101/cloud-security/container-security-checklist/)  
24. OWASP Docker Security Cheat Sheet, accessed September 13, 2025, [https://forums.docker.com/t/owasp-docker-security-cheat-sheet/137987](https://forums.docker.com/t/owasp-docker-security-cheat-sheet/137987)  
25. Resource Management for Pods and Containers \- Kubernetes, accessed September 13, 2025, [https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)  
26. Networking \- Docker Docs, accessed September 13, 2025, [https://docs.docker.com/engine/network/](https://docs.docker.com/engine/network/)  
27. How to Keep Docker Secrets Secure: Complete Guide \- Spacelift, accessed September 13, 2025, [https://spacelift.io/blog/docker-secrets](https://spacelift.io/blog/docker-secrets)  
28. Secrets in Compose \- Docker Docs, accessed September 13, 2025, [https://docs.docker.com/compose/how-tos/use-secrets/](https://docs.docker.com/compose/how-tos/use-secrets/)  
29. How to Manage Secrets in Docker \- freeCodeCamp, accessed September 13, 2025, [https://www.freecodecamp.org/news/manage-secrets-in-docker/](https://www.freecodecamp.org/news/manage-secrets-in-docker/)  
30. Manage sensitive data with Docker secrets, accessed September 13, 2025, [https://docs.docker.com/engine/swarm/secrets/](https://docs.docker.com/engine/swarm/secrets/)  
31. A Comprehensive Guide to Docker Secrets | Better Stack Community, accessed September 13, 2025, [https://betterstack.com/community/guides/scaling-docker/docker-secrets/](https://betterstack.com/community/guides/scaling-docker/docker-secrets/)  
32. Playing with Github API with GO-GITHUB Golang library | by Durgaprasad Budhwani, accessed September 14, 2025, [https://medium.com/@durgaprasadbudhwani/playing-with-github-api-with-go-github-golang-library-83e28b2ff093](https://medium.com/@durgaprasadbudhwani/playing-with-github-api-with-go-github-golang-library-83e28b2ff093)  
33. google/go-github: Go library for accessing the GitHub v3 API \- GitHub, accessed September 14, 2025, [https://github.com/google/go-github](https://github.com/google/go-github)  
34. Creating webhooks \- GitHub Docs, accessed September 13, 2025, [https://docs.github.com/en/webhooks/using-webhooks/creating-webhooks](https://docs.github.com/en/webhooks/using-webhooks/creating-webhooks)  
35. Building a GitHub App that responds to webhook events, accessed September 13, 2025, [https://docs.github.com/en/apps/creating-github-apps/writing-code-for-a-github-app/building-a-github-app-that-responds-to-webhook-events](https://docs.github.com/en/apps/creating-github-apps/writing-code-for-a-github-app/building-a-github-app-that-responds-to-webhook-events)  
36. Webhooks documentation \- GitHub Docs, accessed September 13, 2025, [https://docs.github.com/en/webhooks](https://docs.github.com/en/webhooks)  
37. Events that trigger workflows \- GitHub Docs, accessed September 13, 2025, [https://docs.github.com/actions/learn-github-actions/events-that-trigger-workflows](https://docs.github.com/actions/learn-github-actions/events-that-trigger-workflows)  
38. A Beginner's Guide to Log Management in DevOps \- ELK Stack vs Loki Stack, accessed September 14, 2025, [https://dev.to/yash\_sonawane25/devops-made-simple-a-beginners-guide-to-log-management-in-devops-elk-stack-vs-loki-stack-2f8](https://dev.to/yash_sonawane25/devops-made-simple-a-beginners-guide-to-log-management-in-devops-elk-stack-vs-loki-stack-2f8)  
39. Monitoring & Logging with Prometheus, Grafana, ELK, and Loki (2025 Guide for DevOps), accessed September 14, 2025, [https://www.refontelearning.com/blog/monitoring-logging-prometheus-grafana-elk-stack-loki](https://www.refontelearning.com/blog/monitoring-logging-prometheus-grafana-elk-stack-loki)  
40. Grafana Loki vs. ELK Stack for Logging \- OpsVerse, accessed September 14, 2025, [https://opsverse.io/2024/07/26/grafana-loki-vs-elk-stack-for-logging-a-comprehensive-comparison/](https://opsverse.io/2024/07/26/grafana-loki-vs-elk-stack-for-logging-a-comprehensive-comparison/)  
41. Grafana Loki vs ELK Logging Stacks \- Wallarm, accessed September 14, 2025, [https://www.wallarm.com/cloud-native-products-101/grafana-loki-vs-elk-logging-stacks](https://www.wallarm.com/cloud-native-products-101/grafana-loki-vs-elk-logging-stacks)  
42. Grafana Loki Architecture: A Comprehensive Guide \- DevOpsCube, accessed September 14, 2025, [https://devopscube.com/grafana-loki-architecture/](https://devopscube.com/grafana-loki-architecture/)  
43. Loki overview | Grafana Loki documentation, accessed September 14, 2025, [https://grafana.com/docs/loki/latest/get-started/overview/](https://grafana.com/docs/loki/latest/get-started/overview/)  
44. grafana/loki: Like Prometheus, but for logs. \- GitHub, accessed September 14, 2025, [https://github.com/grafana/loki](https://github.com/grafana/loki)  
45. Promtail vs Fluent Bit: Lightweight Log Shippers for Grafana Loki, accessed September 14, 2025, [https://fluentbit.net/promtail-vs-fluent-bit-which-is-right-for-you/](https://fluentbit.net/promtail-vs-fluent-bit-which-is-right-for-you/)  
46. FluentD and Promtail are both log shippers, each with its strengths and weaknesses. FluentD excels in terms of the number of input and output plugins it offers, coupled with high configuration… \- Dmitrey Kazin \- Medium, accessed September 14, 2025, [https://medium.com/@dmitrey.kazin/fluentd-and-promtail-are-both-log-shippers-each-with-its-strengths-and-weaknesses-daced2a742bf](https://medium.com/@dmitrey.kazin/fluentd-and-promtail-are-both-log-shippers-each-with-its-strengths-and-weaknesses-daced2a742bf)  
47. ECS Logging promtail or fluent-bit : r/aws \- Reddit, accessed September 14, 2025, [https://www.reddit.com/r/aws/comments/12zk0lw/ecs\_logging\_promtail\_or\_fluentbit/](https://www.reddit.com/r/aws/comments/12zk0lw/ecs_logging_promtail_or_fluentbit/)  
48. Fluentd vs Promtail with Loki : r/kubernetes \- Reddit, accessed September 14, 2025, [https://www.reddit.com/r/kubernetes/comments/qv6qqx/fluentd\_vs\_promtail\_with\_loki/](https://www.reddit.com/r/kubernetes/comments/qv6qqx/fluentd_vs_promtail_with_loki/)  
49. Loki architecture | Grafana Loki documentation, accessed September 14, 2025, [https://grafana.com/docs/loki/latest/get-started/architecture/](https://grafana.com/docs/loki/latest/get-started/architecture/)  
50. Loki deployment modes | Grafana Loki documentation, accessed September 14, 2025, [https://grafana.com/docs/loki/latest/get-started/deployment-modes/](https://grafana.com/docs/loki/latest/get-started/deployment-modes/)  
51. Docker vs. containerd vs. Nabla vs. Kata vs. Firecracker and more | by Benjamin | Medium, accessed September 13, 2025, [https://benriemer.medium.com/docker-vs-containerd-vs-nabla-vs-kata-vs-firecracker-and-more-108f7f107d8d](https://benriemer.medium.com/docker-vs-containerd-vs-nabla-vs-kata-vs-firecracker-and-more-108f7f107d8d)  
52. Ignite – Use Firecracker VMs with Docker images | Hacker News, accessed September 13, 2025, [https://news.ycombinator.com/item?id=32990127](https://news.ycombinator.com/item?id=32990127)  
53. Enhancing Kubernetes workload isolation and security using Kata ..., accessed September 13, 2025, [https://aws.amazon.com/blogs/containers/enhancing-kubernetes-workload-isolation-and-security-using-kata-containers/](https://aws.amazon.com/blogs/containers/enhancing-kubernetes-workload-isolation-and-security-using-kata-containers/)  
54. Kata Containers vs Firecracker vs gvisor : r/docker \- Reddit, accessed September 13, 2025, [https://www.reddit.com/r/docker/comments/1fmuv5b/kata\_containers\_vs\_firecracker\_vs\_gvisor/](https://www.reddit.com/r/docker/comments/1fmuv5b/kata_containers_vs_firecracker_vs_gvisor/)  
55. Kata Containers on Kubernetes and Kata Firecracker VMM support | by Gokul Chandra, accessed September 13, 2025, [https://gokulchandrapr.medium.com/kata-containers-on-kubernetes-and-kata-firecracker-vmm-support-28abb3a196e7](https://gokulchandrapr.medium.com/kata-containers-on-kubernetes-and-kata-firecracker-vmm-support-28abb3a196e7)  
56. Please stop saying 'Just use Firecracker' \- do this instead, accessed September 13, 2025, [https://some-natalie.dev/blog/stop-saying-just-use-firecracker/](https://some-natalie.dev/blog/stop-saying-just-use-firecracker/)  
57. Docker Security: 5 Risks and 5 Best Practices for Securing Your Containers \- Tigera, accessed September 14, 2025, [https://www.tigera.io/learn/guides/container-security-best-practices/docker-security/](https://www.tigera.io/learn/guides/container-security-best-practices/docker-security/)  
58. Nix Flake for Reproducible Development — NuttX latest documentation, accessed September 14, 2025, [https://nuttx.apache.org/docs/latest/guides/nix\_flake.html](https://nuttx.apache.org/docs/latest/guides/nix_flake.html)  
59. Flakes \- NixOS Wiki, accessed September 14, 2025, [https://wiki.nixos.org/wiki/Flakes](https://wiki.nixos.org/wiki/Flakes)  
60. Reproducible Development Environments with Nix Flakes | :: aigeruth, accessed September 14, 2025, [https://aige.eu/posts/reproducible-development-environments-with-nix-flakes/](https://aige.eu/posts/reproducible-development-environments-with-nix-flakes/)  
61. Easy development environments with Nix and Nix flakes\! \- DEV Community, accessed September 14, 2025, [https://dev.to/arnu515/easy-development-environments-with-nix-and-nix-flakes-21mb](https://dev.to/arnu515/easy-development-environments-with-nix-and-nix-flakes-21mb)  
62. Have You Heard About Cloud Native Buildpacks? \- DZone, accessed September 14, 2025, [https://dzone.com/articles/have-you-heard-about-cloud-native-buildpacks](https://dzone.com/articles/have-you-heard-about-cloud-native-buildpacks)  
63. My Feedback about Nixpacks \- an alternative to Buildpacks \- Qovery, accessed September 14, 2025, [https://www.qovery.com/blog/my-feedback-about-nixpacks-an-alternative-to-buildpacks/](https://www.qovery.com/blog/my-feedback-about-nixpacks-an-alternative-to-buildpacks/)
