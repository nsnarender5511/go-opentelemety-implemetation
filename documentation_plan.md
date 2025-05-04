# Documentation Plan for Signoz Assignment Project

This document outlines the plan for creating comprehensive documentation for the Signoz Assignment project. The approach follows a bottom-up strategy, starting with foundational components and building context progressively.

## Goal

Create structured, verbose, and interlinked documentation covering:
*   System Architecture
*   Service Functionality
*   Development Setup & Workflow
*   Monitoring & Telemetry (Setup, Key Signals, Dashboards)
*   Include diagrams and visual aids where appropriate.

## Proposed Documentation Structure (`docs/` directory)

```
docs/
├── README.md                 # Overview, entry point
├── Quick Start.md            # Minimal steps to run & observe
├── Glossary.md               # Key terminology definitions
├── architecture/
│   ├── Architecture Overview.md             # High-level overview
│   ├── Service Details.md           # Service details
│   ├── Data Model & Persistence.md         # Data structures (DB, data.json)
│   └── diagrams/             # Placeholder for diagram links
├── features/
│   └── product_service/
│       ├── Product Service Features Overview.md         # Feature overview for product-service
│       ├── Product Service API Endpoints.md  # API details
│       └── Feature: Update Product Stock.md      # Specific feature details (Example)
├── development/
│   ├── Configuration Management.md             # Dev environment setup / Config
│   ├── Building the Services.md           # Build instructions
│   ├── Running Locally with Docker Compose.md    # Running with Docker Compose
│   └── Testing Procedures.md            # Testing procedures (simulator)
├── monitoring/
│   ├── README.md             # Monitoring overview
│   ├── Telemetry Setup.md              # Telemetry config (Go code, OTel collector)
│   ├── SigNoz Dashboards.md         # SigNoz dashboard links/explanations
│   ├── Key Metrics.md        # Important metrics
│   ├── Tracing Details.md             # Tracing details
│   ├── Logging Details.md               # Logging details
│   └── diagrams/             # Placeholder for diagram links
└── assets/                   # Central store for images/diagrams
    ├── images/
    └── diagrams/             # Store for Excalidraw diagrams (.excalidraw or exported .png/.svg)
```

## Standard File Structure (Template for `.md` files)

To ensure consistency and suitability for demo/teaching purposes, all markdown files created within the `docs/` directory will follow this standard structure:

```markdown
# [Page Title]

**Purpose:** [1-2 sentences stating the goal of this page.]
**Audience:** [E.g., Developers, DevOps, Students]
**Prerequisites:** [Optional: Links to prerequisite knowledge/pages. E.g., [[Telemetry Setup]]]
**Related Pages:** [Links to related pages using [[Wikilink]] style, compatible with Obsidian. E.g., [[Other Page]], [[Another Page]]]

---

## 1. Overview & Key Concepts
[High-level summary, essential concepts, terminology, role in the system.]
*   **Concept 1:** [Explanation]
*   **Core Responsibility:** [What does this component *do*?]
*   **Why it Matters:** [Importance in the system/observability.]

---

## 2. Configuration & Setup
[Details on configuration: env vars, config files, code init.]
**Relevant Files:**
*   `path/to/config/file.yaml`
*   `path/to/source/file.go`
**Environment Variables:**
*   `RELEVANT_ENV_VAR`: ...
**Code Initialization:**
```go
// Snippet...
```

---

## 3. Implementation Details & Usage
[How it's implemented, how it's used, code examples.]
**Workflow / Logic:**
1. Step 1
2. Step 2
**Code Examples:**
*   **Calling/Usage:** (`path/to/caller.go`)
    ```go
    // Snippet...
    ```
*   **Internal Logic:** (`path/to/internal.go`)
    ```go
    // Snippet...
    ```

---

## 4. Monitoring & Observability Integration
[Details on logs, traces, metrics integration.]
*   **Logs Emitted:** Key messages, attributes, levels.
*   **Traces:** Span names, attributes, context propagation.
*   **Metrics:** Metric names, types, descriptions, labels.

---

## 5. Visuals & Diagrams
[Embed/link diagrams from `assets/diagrams/`. Explain the diagram.]
![[../assets/diagrams/relevant_diagram.png]]
*Fig 1: Explanation...*

---

## 6. Teaching Points & Demo Walkthrough
[Specific section for education/demo.]
*   **Key Takeaway:** ...
*   **Demo Steps:** 1. Show code... 2. Trigger event... 3. Show SigNoz...
*   **Common Pitfalls / Questions:** ...
*   **Simplification Analogy:** [Optional]

---

## 7. Cross-Cutting Concerns (Optional Section) 
# NEW: Added optional section to template
*   **Error Handling:** [Link to or describe relevant error handling patterns. E.g., [[Error Handling Patterns]] ]
*   **Security Notes:** [Specific security considerations for this component. E.g., Input validation, secrets management.]

---

**Last Updated:** [Date]
```

## Documentation Phases

### Phase 1: Foundational `common` Modules Analysis

*   **Goal:** Understand and document core shared libraries.
*   **How:** Read source code in `common/`.
*   **What & Where:**
    1.  **Glossary (`docs/Glossary.md`):** Define key terms (OTel, Docker, SigNoz, etc.). Document initially and update as needed.
    2.  **Quick Start (`docs/Quick Start.md`):** Outline essential steps to run and see basic observability. Link to other sections for details. Document after core components are understood (perhaps later in Phase 3 or 4).
    3.  **Telemetry Setup (`common/telemetry/setup.go`):** Analyze OTel init, resources, exporters. Document in `docs/monitoring/Telemetry Setup.md`.
    4.  **Logging (`common/log/`):** Analyze custom logger, config, OTel integration. Document in `docs/monitoring/Logging Details.md` and potentially `docs/monitoring/Telemetry Setup.md`.
    5.  **Configuration (`common/config/`):** Analyze config loading mechanism. Document in `docs/development/Configuration Management.md`. **NEW:** Ensure documentation covers key env vars from `docker-compose.yml` (`LOG_LEVEL`, `DATA_FILE_PATH`, etc.) and their effect.
    6.  **Database (`common/db/`):** Analyze DB connection/logic. Document in `docs/architecture/Data Model & Persistence.md` and `docs/development/Running Locally with Docker Compose.md`.
    7.  **Utilities (`common/utils/`):** Analyze general utilities. Mention in relevant feature/architecture docs or `docs/architecture/Architecture Overview.md`.
    8.  **Telemetry Components (`common/telemetry/{attributes,metric,trace,log}/`):** Analyze specific custom signals. Document in `docs/monitoring/Key Metrics.md`, `Tracing Details.md`, `Logging Details.md`.
    9.  **Debug Utilities (`common/debugutils/`):** **NEW:** Analyze purpose and usage. Document in relevant sections (e.g., `Architecture Overview.md`, `Development/Configuration Management.md`).
    10. **Globals (`common/globals/`):** **NEW:** Analyze purpose and usage, especially regarding shared state. Document in relevant sections (e.g., `Architecture Overview.md`).

### Phase 2: Core `product-service` Analysis

*   **Goal:** Understand the main application's structure, features, API, data handling.
*   **How:** Analyze `product-service/src/`, `data.json`, `Dockerfile`.
*   **What & Where:**
    1.  **Service Entrypoint & Structure:** Analyze startup, component init, routing. Document in `docs/architecture/Service Details.md`, `docs/features/product_service/Product Service Features Overview.md`.
    2.  **API Endpoints:** Analyze routes, request/response, logic, telemetry. Document in `docs/features/product_service/Product Service API Endpoints.md`.
    3.  **Business Logic:** Analyze core features. Document in `docs/features/product_service/Feature: Update Product Stock.md`, `docs/architecture/Service Details.md`.
    4.  **Data Handling (`data.json`):** Analyze usage and structure. Document in `docs/architecture/Data Model & Persistence.md`.
    5.  **Docker Setup:** Analyze builds, containers, config, networking. Document in `docs/development/Building the Services.md`, `Running Locally with Docker Compose.md`, `docs/architecture/Service Details.md`. **NEW:** Specifically document the multi-stage build in `product-service/Dockerfile`, explaining each stage and the rationale for the commented-out final stage. **NEW:** Add notes on basic security (network exposure, non-root user pattern importance) in `docs/cross_cutting_concerns/Security Considerations.md` or `docs/development/Running Locally with Docker Compose.md`.
    6.  **Testing (`tests/`, simulator):** Analyze simulator function, interaction, tests. Document in `docs/development/Testing Procedures.md`, `docs/architecture/Service Details.md`. **NEW:** Enhance documentation to detail *how* `simulate_product_service.py` interacts with `product-service` (API endpoints called, nature of traffic).
    7.  **OTel Collector (`otel-collector-config.yaml`):** Analyze receivers, processors, exporters, pipelines, SigNoz integration. Document in `docs/monitoring/Telemetry Setup.md`. **NEW:** Ensure documentation covers key env vars from `docker-compose.yml` (`OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_RESOURCE_ATTRIBUTES`). Add note on securing SigNoz key in `docs/cross_cutting_concerns/Security Considerations.md`.

### Phase 3: Operational Context (Docker, Testing, OTel Collector)

*   **Goal:** Document build, runtime, testing, and external telemetry collection.
*   **How:** Analyze `docker-compose.yml`, `Dockerfile`s, `tests/`, `otel-collector-config.yaml`.
*   **What & Where:**
    1.  **Docker Setup:** Analyze builds, containers, config, networking. Document in `docs/development/Building the Services.md`, `Running Locally with Docker Compose.md`, `docs/architecture/Service Details.md`.
    2.  **Testing (`tests/`, simulator):** Analyze simulator function, interaction, tests. Document in `docs/development/Testing Procedures.md`, `docs/architecture/Service Details.md`.
    3.  **OTel Collector (`otel-collector-config.yaml`):** Analyze receivers, processors, exporters, pipelines, SigNoz integration. Document in `docs/monitoring/Telemetry Setup.md`. **Add note on securing SigNoz key.**

### Phase 4: Monitoring Visualization & Dashboards

*   **Goal:** Describe expected monitoring outputs and link to dashboards.
*   **How:** Conceptualize dashboards based on analysis, obtain SigNoz URLs.
*   **What & Where:**
    1.  **Dashboard Links & Explanations:** Identify key dashboards, get URLs. Document in `docs/monitoring/SigNoz Dashboards.md` (include explanations, potentially **add placeholder screenshots to `assets/images/` initially**).
    2.  **Key Signals Summary:** Consolidate important metrics, traces, logs. Document/refine `docs/monitoring/README.md`, `Key Metrics.md`, `Tracing Details.md`, `Logging Details.md`.
    3.  **Diagrams:** Design and create diagrams using **Excalidraw**. Store source files (`.excalidraw`) or exported images (`.png`, `.svg`) in `docs/assets/diagrams/`. Embed/link these in relevant pages (`docs/architecture/Architecture Overview.md`, `docs/monitoring/README.md`). **NEW:** Specifically plan for:
        *   A **Service Interaction Diagram:** Showing `product-service`, `product-simulator`, `otel-collector` dependencies (`docker-compose.yml`).
        *   A **Telemetry Data Flow Diagram:** Illustrating signal flow: `product-service` -> `otel-collector` -> SigNoz.
    4.  **Overall README (`docs/README.md`):** Write project overview and documentation structure summary. Link to key sections.
    5.  **Linking:** Add `[[Wikilink]]` style links between related pages. Ensure these point to the correct Title Case filenames. This style facilitates easy navigation and integration within an Obsidian vault, potentially managed via the MCP server. **NEW:** Re-emphasize ensuring all relevant cross-links are added during this phase.
    6.  **Review & Refine:** Ensure clarity, consistency, completeness, verbosity. **NEW:** Explicitly check documentation for cross-cutting concerns like error handling and security notes.

### Phase 5: Architecture Synthesis & Final Touches

*   **Goal:** Create high-level diagrams, overviews, and ensure interlinking.
*   **How:** Review docs, create diagrams (Excalidraw), add links.
*   **What & Where:**
    1.  **Diagrams:** Design and create diagrams using **Excalidraw**. Store source files (`.excalidraw`) or exported images (`.png`, `.svg`) in `docs/assets/diagrams/`. Embed/link these in relevant pages (`docs/architecture/Architecture Overview.md`, `docs/monitoring/README.md`).
    2.  **Overall README (`docs/README.md`):** Write project overview and documentation structure summary. Link to key sections.
    3.  **Linking:** Add `[[Wikilink]]` style links between related pages. Ensure these point to the correct Title Case filenames. This style facilitates easy navigation and integration within an Obsidian vault, potentially managed via the MCP server.
    4.  **Review & Refine:** Ensure clarity, consistency, completeness, verbosity.

---
*Plan generated on {current date}* 