**Purpose:** This page describes how application data is structured and persisted.
**Audience:** Developers, DevOps, Students
**Prerequisites:** Basic understanding of JSON.
**Related Pages:** `common/db/file_database.go`, ../../development/Configuration_Management.md, `product-service/data.json`, ./Architecture_Overview.md

---

## 1. Overview & Key Concepts

This application uses a simple file-based persistence mechanism rather than a traditional relational or NoSQL database.

*   **Key Concept: File Database:** A single JSON file acts as the data store.
*   **Key Concept: JSON:** Data is structured and stored in JavaScript Object Notation format.
*   **Core Responsibility:** Provide a mechanism to load application data and save modified data back to the file.
*   **Why it Matters:** Understanding the data persistence method is crucial for knowing how data is loaded, its significant limitations (especially concurrency), and how it relates to application features.

---

## 2. Configuration & Setup

Persistence is managed by the `db.FileDatabase` type defined in `common/db/file_database.go`.

**Relevant Files:**
*   `common/db/file_database.go`: Defines the `FileDatabase` struct and methods.
*   `common/config/config.go`: Provides the file path via configuration.
*   `product-service/data.json`: The actual default data file used.
*   `docker-compose.yml`: Mounts `data.json` into the `product-service` container.

**Configuration:**
*   The path to the JSON data file is determined by the `PRODUCT_DATA_FILE_PATH` configuration value (See ../../development/Configuration_Management.md). The default value is `/app/data.json`.
*   The `docker-compose.yml` file mounts the local `./product-service/data.json` to `/app/data.json` inside the `product-service` container.

**Initialization:**
*   An instance of `FileDatabase` is created using `db.NewFileDatabase()`.
*   This constructor reads the file path from the global configuration (`globals.Cfg().PRODUCT_DATA_FILE_PATH`).

---

## 3. Implementation Details & Usage

The `FileDatabase` provides two main methods for interaction:

**1. `Read(ctx context.Context, dest interface{}) error`**
*   **Functionality:** Reads the entire content of the JSON file specified by `filePath` using `os.ReadFile`.
*   **Concurrency:** This operation itself **does not** use any explicit locking (`sync.Mutex` or `sync.RWMutex`). Concurrent reads are generally safe at the OS level, but this lack of locking is critical when considering write operations.
*   **Usage:** Unmarshals the JSON data into the provided `dest` interface (which should be a pointer to a Go struct matching the JSON structure).
*   **Behavior:** Loads the entire file into memory for processing.

**2. `Write(ctx context.Context, data interface{}) error`**
*   **Functionality:** Writes the provided `data` interface to the JSON file specified by `filePath` using `os.WriteFile`.
*   **Concurrency:** This operation **does not** use any explicit locking (`sync.Mutex` or `sync.RWMutex`).
*   **Usage:** Marshals the `data` into indented JSON format.
*   **Behavior:** Overwrites the *entire* existing file content atomically at the OS level (where supported), but without application-level locking.

**CRITICAL CONCURRENCY ISSUE:**
*   Neither the `FileDatabase` nor the `productRepository` (which uses it) implements application-level locking (`sync.Mutex` or `sync.RWMutex`) around the **read-modify-write cycle** required for operations like `Create` or `UpdateStock`.
*   The typical pattern is:
    1.  Call `FileDatabase.Read()` (no lock).
    2.  Modify the data in memory.
    3.  Call `FileDatabase.Write()` (no lock).
*   If multiple concurrent requests execute this sequence, a **severe race condition** will occur, inevitably leading to **lost updates** and data corruption. Request A might read, Request B reads, Request B writes, then Request A writes its stale data, overwriting B's changes.
*   **Correction:** Previous documentation suggesting the presence of mutexes in either `FileDatabase` or `productRepository` was **incorrect** based on the actual code (`product-service/src/repository.go` and `common/db/file_database.go`). The system currently lacks the necessary protection for concurrent writes.

**Data Structure (`product-service/data.json`)**

The `product-service/data.json` file, used by `FileDatabase`, is expected to contain a **JSON object (map)** where keys are product IDs (strings) and values are objects conforming to the `Product` struct defined in `product-service/src/repository.go`:

```json
// Example structure of data.json
{
  "product-id-1": {
    "id": "product-id-1", // Corrected from ProductID
    "name": "Example Product 1",
    "description": "...",
    "price": 19.99,
    "stock": 100
  },
  "product-id-2": {
    // ... another product ...
  }
}
```

```go
// Go struct defined in product-service/src/repository.go
type Product struct {
    ID          string  `json:"id"` // Matches JSON example
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    Stock       int     `json:"stock"`
}
```
The `FileDatabase.Read` method loads this entire map (`map[string]Product`) into memory. The `productRepository` then interacts with this in-memory map **without adequate locking**.

**Initial File Creation:**
*   The `productRepository`'s `Create` function checks for `os.IsNotExist` when initially reading the file. If the file doesn't exist, it gracefully initializes an empty in-memory product map instead of returning an error, allowing the creation to proceed as the first entry.

**Code Example (Conceptual - Illustrating Lack of Lock):**
```go
// In product-service setup
fileDB := db.NewFileDatabase() // Reads path from config

// Destination map
var products map[string]Product

// Read data (NO LOCK ACQUIRED)
err := fileDB.Read(context.Background(), &products)
// ... handle error ...

// --- Potential concurrent modification by another request starts here ---

// Modify products in memory (NO LOCK HELD)
products["someID"].Stock = newStock

// --- Another concurrent request might read, modify, and WRITE here ---

// Write data back (NO LOCK ACQUIRED)
err = fileDB.Write(context.Background(), products)
// *** If another request wrote between the Read and Write, its changes are lost ***
// ... handle error ...
```

---

## 4. Monitoring & Observability Integration

The `FileDatabase` methods are instrumented for OpenTelemetry tracing:

*   **Traces:**
    *   The `Read` method creates a span with attributes:
        *   `db.system = "file"`
        *   `db.operation = "READ"`
        *   `db.file.path`
    *   The `Write` method creates a span with attributes:
        *   `db.system = "file"`
        *   `db.operation = "WRITE"`
        *   `db.file.path`
    *   Any error occurring during the file operation is recorded on the span.
*   **Logs:** Debug and Error logs are emitted using the shared `slog` logger (`log.L`), including the file path and error details. These logs will be correlated with traces if `otelslog` is active.

---

## 5. Visuals & Diagrams

```mermaid
graph TD
    subgraph Service Layer
        Repo(Product Repository)
    end
    subgraph Common DB Module
        FileDB[db.FileDatabase \n (NO LOCKING)]
    end
    subgraph File System
        JSONFile(data.json)
    end

    Repo -- Calls Read() --> FileDB
    FileDB -- Reads (No Lock) --> JSONFile

    Repo -- Modifies In-Memory --> Repo
    
    Repo -- Calls Write() --> FileDB
    FileDB -- Writes Overwrite (No Lock) --> JSONFile
    
    style FileDB fill:#ffdddd,stroke:#cc0000,stroke-width:2px,color:#cc0000
    style JSONFile fill:#lightgrey,stroke:#333,stroke-width:1px
    linkStyle default interpolate basis stroke:red,stroke-width:2px,color:red;
    
```
*Fig 1: File Database Interaction Flow (**Warning:** Illustrates lack of locking, potential for race conditions).*

---

## 6. Teaching Points & Demo Walkthrough

*   **Key Takeaway:** The application uses a simple JSON file for data storage, accessed via the `FileDatabase`. **Critically, neither `FileDatabase` nor `productRepository` implements the necessary application-level locking (`sync.Mutex`) for concurrent read-modify-write operations.** This makes the current implementation **unsafe for concurrent use** involving writes (`Create`, `UpdateStock`) and will lead to data loss.
*   **Demo Steps:**
    1.  Show `common/db/file_database.go`, highlighting the absence of any `sync.Mutex`.
    2.  Show `product-service/src/repository.go` (`UpdateStock`, `Create`), pointing out the read-modify-write sequence and the lack of any `repo.mu.Lock()` calls.
    3.  Highlight the OTel span creation within `Read`/`Write`.
    4.  Show `product-service/data.json` and explain its map structure.
    5.  Run the application and trigger **multiple concurrent** actions that modify the data (e.g., using `tests/simulate_product_service.py` if it sends concurrent updates, or manually triggering multiple `curl` commands quickly).
    6.  Inspect `data.json` afterwards or query `/products` to demonstrate that updates have likely been lost.
    7.  Explain *why* the data loss occurred due to the race condition.
*   **Common Pitfalls / Questions:**
    *   Is this suitable for production? (**Absolutely not** in its current state due to the concurrency issues).
    *   What happens if two requests try to update stock concurrently? (They will race. One request's update will likely overwrite the other's, leading to incorrect stock levels).
    *   How should this be fixed? (Implement a `sync.Mutex` within the `productRepository` that protects the entire read-modify-write cycle for `Create` and `UpdateStock`).

---

**Last Updated:** 2024-07-30
