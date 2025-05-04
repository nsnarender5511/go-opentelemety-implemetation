

**Purpose:** This page provides a high-level overview of the features provided by the `product-service`.
**Audience:** Developers, Product Managers, Testers, Students
**Prerequisites:** [[Architecture Overview]], [[Architecture Services]]
**Related Pages:** [[Product Service API Endpoints]], [[Feature: Update Product Stock]]

---

## 1. Core Functionality

The `product-service` is responsible for managing product information stored in a file-based data store. Its primary features include:

*   **Retrieving Products:** Fetching a list of all available products or a single product by its ID.
*   **Creating Products:** Adding new products to the catalog with basic validation and unique ID generation.
*   **Updating Stock:** Modifying the stock level for a specific product.
*   **Health Checks:** Providing endpoints to check the service status.

---

## 2. Feature Breakdown

Detailed descriptions of the API endpoints implementing these features can be found in [[Product Service API Endpoints]].

*   **Get All Products:** Retrieves all products currently loaded from `data.json`.
*   **Get Product by ID:** Retrieves a specific product matching the provided ID from the in-memory map.
*   **Create Product:** Validates input (name required, price/stock >= 0), generates a unique product ID (using `uuid`), adds the new product to the in-memory map, and triggers a write back to `data.json` (requires locking).
*   **Update Product Stock:** [[Feature: Update Product Stock]] - Modifies the stock count for an existing product, triggering a write back (requires locking).
*   **Health/Status Check:** Simple endpoints (`/health`, `/status`) returning `200 OK` to indicate the service is running.

---

**Last Updated:** [Current Date]
