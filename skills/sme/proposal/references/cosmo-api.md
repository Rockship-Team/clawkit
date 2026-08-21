# COSMO API Reference

## Endpoints

### Search Contacts
```
POST /v2/contacts/search
Body: {"query": "company_name", "pageSize": 25}
Response: { "data": { "total": N, "list": [...] } }
```

### Get Contact by ID
```
POST /v1/contacts/{id}
Response: { "data": { "entity": { ... } } }
```

### Create Contact
```
POST /v1/contacts
Body: { "name": "...", "company": "...", ... }
Response: { "data": { "id": "uuid", ... } }
```

### Get Interactions
```
GET /v1/contacts/{id}/interactions?limit=50
Response: { "data": [ { ... } ] }
```

### Log Interaction
```
POST /v1/contacts/{id}/interactions
Body: { "type": "...", "timestamp": "ISO8601", "created_by": "system" }
Response: { "status": "ok" }
```

## Authentication
- **Header:** `Authorization: Bearer {COSMO_API_KEY}`
- **Base URL:** COSMO_BASE_URL environment variable (default: http://localhost:8081)

## Error Handling
- **401:** Token expired or invalid
- **404:** Contact not found
- **400:** Invalid request body
- **500:** Server error

## Rate Limits
- **Requests per minute:** 100
- **Recommendation:** Batch requests when possible
