# aws-tui

`aws-tui` is an interactive Terminal User Interface (TUI) for managing AWS infrastructure, heavily inspired by the workflow, speed, and aesthetics of `k9s`. It aims to provide developers and cloud administrators with a fast, keyboard-centric way to list, view, describe, delete, and operate on diverse AWS resources without navigating the web console or typing lengthy AWS CLI commands.

## Key Concepts

`aws-tui` is built on a few core abstractions that keep the codebase lightweight and highly modular:

### 1. Core TUI Engine
The orchestrator of the terminal screen. Built on top of `tview` and `tcell`, it handles:
* Keyboard routing (e.g., typing `:` opens a command palette to switch resources, similar to `k9s`).
* AWS Session and profile management (supporting IAM roles, SSO, MFA, and region switching).
* State management for the active screen (Tables, YAML Describe views, Log streams).
* Background worker execution, preventing the UI from freezing during network requests.

### 2. Resource Providers
A **Resource Provider** is a self-contained module that handles the lifecycle of a specific AWS resource type (e.g., EC2, S3, RDS, Lambda). It defines:
* How to fetch and query instances of the resource from the AWS API.
* How to format the resource attributes into table columns.
* Which custom commands/hotkeys are supported (e.g., hitting `s` on an EC2 instance initiates an SSM SSH session; hitting `l` on a Lambda function tails its CloudWatch log stream).

### 3. Unified View Mapping
Instead of each AWS resource module rendering its own tables and views, they implement standard interfaces. The **Core TUI Engine** dynamically renders these outputs using generic UI components (`TableView`, `DescribeView`, `LogView`). This guarantees a unified aesthetic and keyboard shortcut experience across all services.

### 4. Dynamic vs. Compile-time Extensibility
Because AWS contains hundreds of distinct resource types, the project adopts a **compile-time registration** model as its primary modularity vector:
* Developers can easily add support for a new AWS resource by implementing a single Go interface and registering it via an `init()` block.
* For dynamic out-of-tree plugins, the architecture supports a gRPC-based provider wrapper (inspired by HashiCorp's plugin system), allowing external processes to register new resource types over a local socket.

---

## Technical Architecture

For a deep dive into the design decisions, interface definitions, Go framework choices, and directory layouts, refer to [docs/architecture.md](file:///home/sealekse/GolandProjects/aws-tui/docs/architecture.md).

## Project Structure

* [cmd/aws-tui/](file:///home/sealekse/GolandProjects/aws-tui/cmd/aws-tui/): The entry point and configuration bootstrapping.
* [internal/tui/](file:///home/sealekse/GolandProjects/aws-tui/internal/tui/): The core TUI application, routing, layout rendering, and event handlers.
* [pkg/providers/](file:///home/sealekse/GolandProjects/aws-tui/pkg/providers/): Concrete resource handlers (EC2, S3, RDS, Lambda, etc.).
* [pkg/provider/](file:///home/sealekse/GolandProjects/aws-tui/pkg/provider/): Core provider interface definitions and registration utilities.
* [pkg/awsclient/](file:///home/sealekse/GolandProjects/aws-tui/pkg/awsclient/): AWS API wrappers, authentication session helpers, and credential helpers.