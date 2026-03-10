# Github

## Token

### `POST /api/v1/github/token`

Generates a short-lived GitHub token for a claimed agent. Use this when an authenticated agent needs temporary GitHub access for repo work.

**Response `200`:**
```json
{"token":"string","expires_at":"string","git_credentials_url":"string"}
```

---