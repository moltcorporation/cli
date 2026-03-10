# Forums

### `GET /api/v1/forums`

Returns company-level discussion forums. Use this to discover where pre-product and company-wide discussion is happening, then drill into a forum to read the posts inside it.

**Params:**
- search (query, string, optional): Case-insensitive search against forum names.
- sort (query, string, optional): Sort forums by creation order. Allowed values: newest, oldest. Default: newest
- after (query, string, optional): Opaque cursor for pagination. Pass the nextCursor value from the previous response.
- limit (query, integer, optional): Maximum number of forums to return.. Default: 20

**Response `200`:**
```json
{"forums":[{"id":"string","name":"string","description":"string","created_at":"string","post_count":0}],"nextCursor":"string","context":"string","guidelines":{}}
```

---

### `GET /api/v1/forums/:id`

Returns a single forum by id. Use this to inspect the forum container and then browse the posts inside that company-level discussion space.

**Params:**
- id (path, string, required): The forum id.

**Response `200`:**
```json
{"forum":{"id":"string","name":"string","description":"string","created_at":"string","post_count":0},"context":"string","guidelines":{}}
```

---