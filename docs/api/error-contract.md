# Error contract

## Implemented

```json
{"error":{"code":"unavailable","message":"Service unavailable"}}
```

| Status | Code | Public message | Trigger/evidence |
| --- | --- | --- | --- |
| 404 | `not_found` | Not found | Router fallback; `tests/http.rs` |
| 405 | `method_not_allowed` | Method not allowed | Known-route unsupported method; `tests/http.rs` |
| 503 | `unavailable` | Service unavailable | Failed/timed-out readiness; HTTP and pool-saturation tests |
| 500 | `internal_error` | Internal server error | Fixed `HttpError::Internal` response tested; no product handler currently emits it |

Health success is `{"status":"ok"}`. Router responses have no-store/nosniff;
normal HEAD behavior suppresses response bodies. No raw DB error or credential
appears in public errors. The migration CLI exits nonzero with fixed diagnostic
text on failure; its output is not an HTTP response.

## Planned product extensions

Authentication (`401`), access policy (`403` or concealed `404`), field validation
(`422`), stale edits (`409`), request limits (`413`) and abuse limits (`429`) need
explicit feature contracts and tests. Proposed field errors/correlation metadata
must preserve stable codes and avoid secrets/private content. They are not yet
implemented merely because these status codes are listed here.
