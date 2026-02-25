# Responsive Design — ATLinks

**Date:** 2026-02-25
**Status:** Approved
**Scope:** UX-1 (HIGH) — Add mobile/responsive breakpoints

## Context

ATLinks is a desktop-first logistics app with zero screen-size media queries. The nav, tables, filter bars, and form grids break below ~1100px. Primary mobile users are **drivers** looking up trips/loads on their phones. Secondary users are **dispatchers** accessing the app remotely for convenience.

## Approach

**Option B — Practical responsive** (chosen over minimal scroll-only or full mobile-first rewrite)

Two breakpoints added to `internal/handler/static/css/app.css`:
- `@media (max-width: 768px)` — mobile phones
- `@media (max-width: 1024px)` — tablets

No new dependencies. All changes are CSS-only; no template logic changes required.

## Design

### Nav (≤1024px)
- Full horizontal menu hidden
- Hamburger button (☰) added to right side of nav bar
- Tapping opens full-width vertical dropdown with all menu items
- Alpine.js handles open/close toggle (already used in nav for dropdowns)
- Brand name stays visible at all times

### Tables
- `.table-container` already has `overflow-x: auto` — audit templates to ensure all tables are wrapped
- Tables scroll horizontally on mobile rather than breaking layout

### Forms & Filter Bars
- `.form-row` already uses `auto-fit minmax(200px, 1fr)` — stacks naturally
- Filter bars get `flex-direction: column` on mobile so each input is full width
- `.container` padding tightened from `20px` to `12px` on mobile

### Other Components
- `.page-header` switches to column layout on mobile (title above buttons)
- `.form-actions` buttons go full-width on mobile
- `.feedback-modal` width changes from fixed `380px` to `calc(100vw - 48px)` on mobile
- Dashboard/KPI grids already use `auto-fit` — stack naturally

## Testing

All changes verified locally at `localhost:8080` before deploying to `atlinks.app`.
