# Reference guide for normalizing client data into structured briefs

## Output Structure

```markdown
# Client Brief: {Company}

## 1. Company Profile
- **Name:** {legal_name}
- **Industry:** {industry}
- **Size:** {employee_count} employees, {revenue} revenue
- **Location:** {hq_address}
- **Business Model:** {description}

## 2. Contact Information
- **Primary Contact:** {name}, {title}
- **Email:** {email}
- **Phone:** {phone}
- **Decision Maker:** {yes/no}

## 3. Requirements Summary
### Business Objectives
- {objective_1}
- {objective_2}
- {objective_3}

### Pain Points
- {pain_point_1}
- {pain_point_2}
- {pain_point_3}

### Desired Outcomes
- {outcome_1}
- {outcome_2}
- {outcome_3}

## 4. Technical Requirements
- **Current Stack:** {list}
- **Integration Needs:** {list}
- **Deployment:** {cloud/on-premise}
- **Compliance:** {requirements}

## 5. Commercial Terms
### Budget
- **Range:** {min}-{max}
- **Source:** {internal/external}
- **Timeline:** {approval_date}

### Contract Terms
- **Duration:** {months} months
- **Payment Terms:** {terms}
- **Decision Process:** {steps}

## 6. Signals & Indicators
### Positive Signals
- [ ] {signal_1}
- [ ] {signal_2}

### Red Flags
- [ ] {flag_1}

## 7. Missing Information (TBD)
- [ ] {item_1}
- [ ] {item_2}
- [ ] {item_3}

## 8. Meeting Notes
{raw meeting notes converted to structured bullets}

## 9. Recommended Proposal Type
{ai-agent | consulting | custom-dev | saas | managed-services}

**Rationale:** {reason based on requirements}
```

## Normalization Rules

### Data Sources Hierarchy
1. **COSMO CRM** (highest priority)
2. **Apollo.io** (secondary)
3. **User Input** (tertiary)

### Field Mapping
| Source Field | Output Field | Example |
|-------------|--------------|---------|
| `entity.name` | `Contact Name` | "Khoa Account 3" |
| `entity.company` | `Company Name` | "Heineken Vietnam" |
| `entity.job_title` | `Title` | "Founder" |
| `ai_insights` | `Signals` | extracted from AI analysis |

### Budget Format
- Normalize all to: `{currency} / {period}`
- Convert ranges to: `{min} - {max}`
- Flag if undefined: `TBD`

### Timeline Format
- Use relative: `{weeks} weeks`, `{months} months`
- Absolute dates: `YYYY-MM-DD`
- If unknown: `TBD`

## Common Patterns

### AI Agent Requirements
- **Keywords:** "chatbot", "automation", "AI", "customer service"
- **Typical Budget:** 15-400M VND/year
- **Scale:** 1,000-50,000 conversations/month
- **Channels:** Website, Zalo, Email, SMS

### Consulting Engagements
- **Keywords:** "strategy", "optimization", "audit", "consulting"
- **Typical Budget:** 50-800M VND/project
- **Duration:** 2-12 weeks
- **Deliverables:** Report, recommendations, implementation plan

### Custom Development
- **Keywords:** "build", "develop", "custom", "software"
- **Typical Budget:** 200M-2B VND
- **Duration:** 3-12 months
- **Features:** Detailed feature list required

### SaaS subscriptions
- **Keywords:** "subscription", "platform", "SaaS", "monthly"
- **Typical Budget:** 10-400M VND/year
- **Duration:** 12+ months
- **Scale:** Users/transactions/month

### Managed Services
- **Keywords:** "support", "maintenance", "managed", "SLA"
- **Typical Budget:** 5-100M VND/month
- **Duration:** 12+ months
- **Services:** Monitoring, helpdesk, security, backups

## Quality Checks

### Before Finalizing Brief
- [ ] All required fields populated or marked TBD
- [ ] Budget range specified or flagged
- [ ] Timeline estimated or marked TBD
- [ ] Decision maker identified
- [ ] Competitive context noted (if any)
- [ ] Missing info clearly listed

### After COSMO Lookup
- [ ] Check for duplicate contacts
- [ ] Pull ai_insights for additional context
- [ ] Review past interactions (if any)
- [ ] Verify company/industry classification

### After Apollo Lookup
- [ ] Enrich with employee count
- [ ] Enrich with funding stage
- [ ] Add recent news/mentions
- [ ] Identify key decision makers

## Edge Cases

### No COSMO Data
→ Use user-provided information
→ Flag as "New Lead - Manual Data Entry"
→ Skip ai_insights

### Incomplete Meeting Notes
→ Extract what's available
→ List all missing requirements
→ Recommend discovery call

### Conflicting Budget Info
→ Note both figures
→ Flag for clarification
→ Provide range in proposal

### Multiple Decision Makers
→ List all with roles
→ Identify primary contact
→ Note escalation path
