# Project Agents

This directory contains specialized agent prompts for reviewing and maintaining code quality in this project.

## Available Agents

### 🔵 go-lead
**Focus:** Go code quality, idioms, and performance
**Expertise:** Uber Go Style Guide, stdlib patterns, best practices

**When to use:**
- After implementing new features
- Before committing significant Go code changes
- When unsure about Go idioms or patterns
- For performance-critical code reviews

**Usage:**
```
@go-lead Please review the shipment handler implementation
```

---

### 🔒 tenant-guardian
**Focus:** Multi-tenant data isolation and security
**Expertise:** Preventing cross-tenant data leaks

**When to use:**
- **CRITICAL:** After ANY SQL query changes
- After implementing or modifying data access handlers
- Before committing database-related code
- Before production deployments

**Usage:**
```
@tenant-guardian Review all shipment queries for tenant isolation
```

**Note:** This is the most critical agent for this application. One missed `organization_id` check = security breach.

---

### 🛡️ security-auditor
**Focus:** Web security, authentication, authorization
**Expertise:** OWASP Top 10, input validation, CSRF, XSS

**When to use:**
- After implementing auth/authorization
- Before production deployment
- After adding user input handling
- When implementing file uploads
- For general security reviews

**Usage:**
```
@security-auditor Audit the authentication implementation
```

---

### 🎨 htmx-architect
**Focus:** HTMX patterns, Templ best practices, UI/UX
**Expertise:** Hypermedia patterns, progressive enhancement

**When to use:**
- After implementing new UI features
- When adding HTMX interactions
- After creating or modifying Templ templates
- For UX improvements
- When reviewing frontend patterns

**Usage:**
```
@htmx-architect Review the shipment list component
```

---

## Recommended Workflow

### For New Features

1. **Implement the feature** (write code)
2. **@tenant-guardian** - Check data isolation (if touching database)
3. **@go-lead** - Review Go code quality
4. **@htmx-architect** - Review frontend patterns (if UI changes)
5. **@security-auditor** - Security check before commit

### Before Production Deployment

1. **@tenant-guardian** - Full audit of all queries
2. **@security-auditor** - Complete security audit
3. **@go-lead** - Code quality review
4. **@htmx-architect** - UX/accessibility review

### Quick Checks

**After database changes:**
```
@tenant-guardian Review all new queries in sql/queries/
```

**After handler changes:**
```
@go-lead Review handlers in internal/handlers/
@tenant-guardian Verify organization_id is used in all queries
```

**After template changes:**
```
@htmx-architect Review templates in templates/
```

## Agent Priority by Risk

**Critical (Always use):**
1. 🔒 **tenant-guardian** - Data leak prevention
2. 🛡️ **security-auditor** - Security vulnerabilities

**Important (Use frequently):**
3. 🔵 **go-lead** - Code quality and maintainability

**Recommended (Use as needed):**
4. 🎨 **htmx-architect** - UX and frontend patterns

## Example Usage

### Scenario 1: Adding a new CRUD feature

```markdown
I just implemented the customer CRUD feature. Can you review it?

Files changed:
- sql/queries/customers.sql
- internal/handlers/customers.go
- templates/customers.templ

@tenant-guardian Please verify all queries include organization_id filtering
@go-lead Review the handler implementation for Go best practices
@htmx-architect Review the Templ templates for HTMX patterns
```

### Scenario 2: Before committing auth changes

```markdown
I implemented user authentication with sessions. Ready to commit?

@security-auditor Please audit:
- Password hashing (internal/auth/password.go)
- Session management (internal/auth/session.go)
- Login handler (internal/handlers/auth.go)

@tenant-guardian Verify organization isolation in auth flow
```

### Scenario 3: Quick query check

```markdown
Added a new query to shipments.sql - does it look safe?

@tenant-guardian Review the GetShipmentsByStatus query at line 45
```

## Tips for Effective Reviews

1. **Be specific** - Point agents to exact files/functions
2. **Provide context** - Explain what the code does
3. **Ask targeted questions** - "Does this prevent X?" vs "Review this"
4. **Use multiple agents** - Different perspectives catch different issues
5. **Review incrementally** - Don't wait until everything is done

## Agent Customization

These agents are maintained in this directory. To customize:

1. Edit the respective `.md` file
2. Add project-specific patterns or rules
3. Update examples with actual code from this project
4. Commit changes so all team members use updated agents

## Contributing

When you discover common issues or patterns:

1. Document them in the appropriate agent file
2. Add examples from real code reviews
3. Update the checklist items
4. Share learnings with the team

---

**Remember:** Agents are here to help maintain quality and catch issues early. Use them liberally - they're much cheaper than production bugs!
