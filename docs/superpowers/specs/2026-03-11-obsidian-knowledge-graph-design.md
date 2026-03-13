# ATLinks Obsidian Knowledge Graph — Design

**Date:** 2026-03-11
**Purpose:** Build a comprehensive Obsidian knowledge graph for ATLinks, optimized for Claude+Brady collaboration across sessions. Serves as extended memory, decision log, and troubleshooting reference.

## Audience & Goals

- **Audience:** Brady + Claude only (not shared externally)
- **Primary goal:** Prevent context/lesson loss as the app grows more complex
- **Secondary goal:** Speed up session starts — Claude can search vault for context instead of re-exploring code

## Approach: Hub and Spoke

Central hub note linking to focused topic notes. Each note is self-contained and searchable via frontmatter tags.

## Structure

```
work/projects/atlinks/
├── ATLinks Hub.md              # Central index
├── architecture/
│   ├── Stack & Structure.md    # Tech stack, project layout, key packages
│   ├── Domain Model.md         # Entity relationships, business rules
│   └── Deployment.md           # Infra, deploy pipeline, NAS details
├── features/
│   ├── Auth & Multi-tenancy.md
│   ├── Dispatch.md             # Orders, VINs, trips, loads
│   ├── Accounting.md           # Invoices, payments, credit memos, AP
│   ├── Loadboard.md
│   ├── QBO Integration.md
│   ├── Reports & Dashboard.md
│   └── Feedback & Notifications.md
├── decisions/
│   └── Decisions Index.md      # Table of all decisions + links
├── lessons/
│   └── Lessons Index.md        # Template-ready, populated over time
└── Roadmap.md                  # Known future work, priorities
```

Existing MCP notes (12 notes in `work/projects/`) stay in place — Hub links into them.

## Conventions

- **Frontmatter:** `tags: [atlinks, <category>]`, `created`, `type`
- **Wikilinks:** `[[Note Name]]` for all cross-references
- **Feature note template:** What It Does → Key Files → How It Works → Gotchas → Related
- **Decisions:** Simple ones as rows in index table; complex ones get their own note
- **Lessons:** One per incident/insight, added as they occur

## Note Count

~15 new notes + linking to 12 existing MCP notes = ~27 total in the graph.
