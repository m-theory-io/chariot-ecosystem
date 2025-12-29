**Add Arrow/Parquet Targets**

- **Target abstraction**: extend `doETL`’s `targetConfig` to accept a new `type` (e.g., `parquet`). Keep existing SQL/Couchbase code untouched by routing through a `target := newTargetWriter(config)` factory.
- **Arrow writer**: use Apache Arrow Go libs to build the schema from transform mappings, feed rows as they stream through transformations, and flush to a `memory.Buffer`. When the job completes, serialize to Parquet (Arrow provides a Parquet writer) so we don’t build our own columnar representation.
- **Destination adapters**: after obtaining the final Parquet bytes, plug in a “sink” layer that handles:
  1. Local filesystem (default folder from config, respect sandbox rules).
  2. Azure Blob Storage (use existing secret management to grab credentials, rely on the Azure Blob Go SDK).
  3. S3-compatible storage (AWS SDK or minio client, credentials resolved via secrets).
  Each sink just implements `Write(ctx, path, reader)` so new cloud targets can be added later.
- **Config**: expand `targetConfig` to include `destination` (local|azure|s3), `path` (folder/blob prefix), and relevant credentials (or named secret). For cloud destinations, prefer referencing vault keys so configs stay declarative in agents/onStart scripts.
- **Runtime scope**: Arrow/Parquet writing can happen inside the ETL goroutine, but heavy jobs might need chunking—stream rows into Arrow record batches (e.g., 5–10k rows), flush to Parquet incrementally, and upload once after closing the writer.
- **Permissions**: ensure sandbox paths are validated (reuse `getSecureFilePath`), and block direct cloud uploads unless `sandbox_enabled=false` or the agent explicitly grants it via policy.
- **Testing**: create integration tests that (a) write to a temp folder and inspect the Parquet schema with Arrow readers, (b) stub Azure/S3 clients so we can assert the right blob/key names are used.

This approach keeps the current ETL API stable while adding a modular output path for columnar formats and remote storage.