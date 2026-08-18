# Lossless Passthrough and Canonical Idempotency

## Problem Statement

The contract requires `metadata` to be preserved verbatim for audits and disputes. Decoding arbitrary JSON into `map[string]any` converts numbers to `float64`, which can corrupt large integers. Raw bytes alone, however, are unsuitable for stable duplicate comparison because harmless whitespace and object-key order differ across retries.

## Options Evaluated

### Option A: Parsed values only

| Pros | Cons |
| --- | --- |
| Simple access and marshaling | Loses numeric precision and original producer representation |

### Option B: Raw values only

| Pros | Cons |
| --- | --- |
| Preserves exact evidence | Whitespace and key order make equivalent retries look different; structured consumers must parse repeatedly |

### Option C: Dual raw and canonical representations

| Pros | Cons |
| --- | --- |
| Preserves audit bytes and gives stable, number-safe duplicate comparison | Adds storage fields and explicit boundary translation |

## Decision

Choose Option C. The HTTP boundary retains raw `location` and `metadata` JSON. The domain stores independent byte copies, while fingerprinting uses a deterministic representation that never converts numbers through binary floating point.

## Implementation Details

- Use `json.RawMessage` for arbitrary producer objects and validate their JSON object shape.
- Canonicalize recursively with `json.Decoder.UseNumber`, sorted object keys, and exact numeric lexemes.
- Persist raw bytes in NDJSON, PostgreSQL binary/text columns, and DynamoDB binary attributes.
- Keep the idempotency key `(source, source_reference)` and use the fingerprint only to distinguish a replay from conflicting content.
- Maintain backward reads for records that predate raw fields.

