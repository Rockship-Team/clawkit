# COSMO Common Workflows — Quick Reference

Use `scripts/cosmo_api.sh` for all calls. Replace UUID with actual IDs.

## Create Campaign for Specific Contacts (most common)

Five steps, in order:

```bash
# Step 1: Get agent ID (email sender account)
scripts/cosmo_api.sh POST /v1/agents/search '{"filter_":{}}'
# → note the agent id from response

# Step 2: Create contact list with contacts already attached
scripts/cosmo_api.sh POST /v1/list-contacts '{"name":"LIST_NAME","contact_ids":["CONTACT_UUID_1","CONTACT_UUID_2"]}'
# → note the list id from response

# Step 3: Create campaign as DRAFT first (status must be "draft", NOT "active")
scripts/cosmo_api.sh POST /v1/campaigns '{"name":"CAMPAIGN_NAME","playbook":"PLAYBOOK","list_contact_id":"LIST_UUID","agent_id":"AGENT_UUID","status":"draft"}'
# → note the campaign id from response

# Step 4: Generate AI email templates
scripts/cosmo_api.sh POST /v3/campaigns/CAMPAIGN_UUID/templates

# Step 5: Activate campaign (THIS triggers the worker to actually send emails)
scripts/cosmo_api.sh PATCH /v1/campaigns/CAMPAIGN_UUID '{"status":"active"}'
# → emails will now be queued and sent by the background worker
```

⚠️ **IMPORTANT**: Creating a campaign with status="active" does NOT trigger email sending.
You MUST create as "draft" first, then PATCH to "active" — only the status CHANGE triggers the worker.

Playbook options: `cold_outreach`, `revive_dormant_leads`, `upsell_existing_customers`, `event_invite`, `content_offering`, `webinar_follow_up`

## Find a Contact

```bash
scripts/cosmo_api.sh POST /v2/contacts/search '{"query":"NAME_OR_COMPANY","pageSize":5}'
# → contact data is in .data.list[].entity
```

## Create an Event

```bash
scripts/cosmo_api.sh POST /v1/events '{"title":"EVENT_TITLE","date":"2026-05-01T14:00:00+07:00","venue":"VENUE","address":"ADDRESS","capacity":20,"status":"published","time_display":"2:00 PM - 4:30 PM","metadata":{"schedule":[{"time":"14:00","title":"Intro","description":"..."}],"takeaways":["item1"],"audience":["item1"],"preparation":["item1"]},"external_urls":{"telegram_group":"URL","luma_url":"URL"}}'
```

Public page will be at: `/events/{auto-generated-slug}`

## Enrich a Contact with AI

```bash
scripts/cosmo_api.sh POST /v1/contacts/CONTACT_UUID/enrich
```

## Log an Interaction

```bash
scripts/cosmo_api.sh POST /v1/interactions '{"contact_id":"UUID","type":"TYPE","channel":"CHANNEL","direction":"DIRECTION","content":"DESCRIPTION"}'
```

- type: `email`, `call`, `meeting`, `linkedin_message`, `note`, `proposal_sent`
- channel: `Email`, `Phone`, `LinkedIn`, `InPerson`
- direction: `inbound`, `outbound`

## Schedule a Meeting

```bash
scripts/cosmo_api.sh POST /v1/outreach/meetings '{"contact_id":"UUID","title":"TITLE","time":"ISO_DATE","channel":"google_meet","location":"URL_OR_ADDRESS"}'
```

## Update Outreach State

States: `COLD` → `NO_REPLY` → `REPLIED` → `POST_MEETING` → `DROPPED`

```bash
scripts/cosmo_api.sh POST /v1/outreach/UUID/update '{"conversation_state":"REPLIED"}'
```

## Generate AI Email Templates for a Campaign

```bash
scripts/cosmo_api.sh POST /v3/campaigns/CAMPAIGN_UUID/templates
# → generates personalized email templates using AI
```

## Search Knowledge Base (for RAG)

```bash
scripts/cosmo_api.sh POST /v1/knowledge/search '{"query":"SEARCH_QUERY","limit":5}'
```

## Key Rules

- Always search for contact FIRST before creating (avoid duplicates)
- Always get agent_id from `/v1/agents/search` before creating campaigns
- Create contact list WITH contact_ids in one step (don't create empty then patch)
- ALWAYS create campaign as `"draft"` first, then PATCH to `"active"` — creating with status="active" does NOT trigger sending
- The PATCH status change from draft→active is what enqueues the campaign worker
- Contact IDs are UUIDs — get them from search results `.data.list[].entity.id`
