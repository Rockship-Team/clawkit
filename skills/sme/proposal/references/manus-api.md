# Manus AI API Reference

## Authentication
- **Header:** `API_KEY: {MANUS_API_KEY}`
- **Base URL:** `https://api.manus.ai` (override via `MANUS_BASE_URL`)

## Endpoints

### Create Task
```
POST /v1/tasks
Headers: API_KEY: {MANUS_API_KEY}, Content-Type: application/json
Body: {
  "prompt": "task instructions",
  "agentProfile": "manus-1.6",        // options: manus-1.6, manus-1.6-lite, manus-1.6-max
  "attachments": [],                   // optional: file IDs, URLs, or base64
  "taskMode": "agent",                 // options: chat, adaptive, agent
  "locale": "en-US",                   // optional
  "createShareableLink": false         // optional
}
Response: {
  "task_id": "...",
  "task_title": "...",
  "task_url": "...",
  "share_url": "..."
}
```

### Get Task (poll for completion)
```
GET /v1/tasks/{task_id}
Headers: API_KEY: {MANUS_API_KEY}
Query: ?convert=false
Response: {
  "id": "...",
  "status": "pending|running|completed|failed",
  "metadata": { "task_title": "...", "task_url": "..." },
  "output": [
    {
      "role": "assistant",
      "content": [
        {
          "type": "output_text",
          "text": "...",
          "fileUrl": "https://...",     // PDF download URL
          "fileName": "proposal.pdf",
          "mimeType": "application/pdf"
        }
      ]
    }
  ],
  "credit_usage": 245
}
```

### List Tasks
```
GET /v1/tasks
Headers: API_KEY: {MANUS_API_KEY}
Response: { "tasks": [...] }
```

### Upload File
```
POST /v1/files
Headers: API_KEY: {MANUS_API_KEY}
Body: multipart/form-data with file
Response: { "id": "file_...", "url": "..." }
```

## Task Status Flow
`pending` → `running` → `completed` | `failed`

## Agent Profiles
| Profile | Use case |
|---------|----------|
| `manus-1.6-lite` | Simple tasks, lower cost |
| `manus-1.6` | Default, balanced |
| `manus-1.6-max` | Complex multi-step tasks, highest quality |

## Rate Limits
- Check Manus dashboard for current limits
- Recommendation: Use `manus-1.6` for proposals, `manus-1.6-max` for complex ones

## Scripts
- `scripts/manus_create_task.sh` — pipe JSON body via stdin
- `scripts/manus_get_task.sh <task_id>` — polls until completed (5 min timeout)
