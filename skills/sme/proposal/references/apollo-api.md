# Apollo.io API Reference

## Endpoints

### Search Company
```
GET https://api.apollo.io/v1/companies/follow?q={company_name}
Headers: Authorization: Bearer {APOLLO_IO_API_KEY}
Response: { "companies": [ { ... } ] }
```

### Search People
```
GET https://api.apollo.io/v1/people/search?q={company_name}&seniorities={seniority_list}
Headers: Authorization: Bearer {APOLLO_IO_API_KEY}
Response: { "people": [ { ... } ] }
```

### Enrich Person
```
GET https://api.apollo.io/v1/people/enrich?first_name={fn}&last_name={ln}&company_domain={domain}
Headers: Authorization: Bearer {APOLLO_IO_API_KEY}
Response: { "person": { ... } }
```

## Authentication
- **API Key:** Set `APOLLO_IO_API_KEY` environment variable
- **Base URL:** `https://api.apollo.io`

## Data Enrichment Fields
- Employee count
- Funding stage
- Industry classification
- Recent news mentions
- Social profiles

## Rate Limits
- **Requests per day:** 5,000
- **Recommendation:** Cache results
