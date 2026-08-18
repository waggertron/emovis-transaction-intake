# BUG-2026-08-18-002 — Missing Required Fields Accepted

**Status:** Fixed on `feat/openapi-contract-conformance`

## Observed behavior

The HTTP decoder represented missing strings as empty Go strings and delegated validation to the application. Transport tests using an intake fake returned `201` when `source`, `source_reference`, `transaction_type`, or `base_amount` was omitted. Domain validation also silently defaulted a missing type to `toll`.

## Expected behavior

Every field listed in `IngestRequest.required` must be present and string-shaped. A missing `transaction_type` must not be compiled or defaulted into the request.

## Root cause

The request DTO could not distinguish missing, null, and empty values, and legacy migration defaults remained in domain validation.

## Correction

The HTTP adapter decodes an object into named raw fields, explicitly requires all five properties, and the domain rejects an empty or unconfigured transaction type.

## Validation evidence

`TestIngestRejectsEveryMissingRequiredProperty` and `TestTransactionRequiresExplicitConfiguredType` cover the regression.

