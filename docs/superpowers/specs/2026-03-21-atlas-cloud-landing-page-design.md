# Atlas Cloud Landing Page Design

**Date**: 2026-03-21
**Status**: Approved

## Overview

Add a public marketing landing page for atlascloud.app. Unauthenticated visitors see the landing page; authenticated users go straight to the dashboard. The landing page is a standalone HTML page using Tailwind CDN, not the app's existing CSS system.

## Routing

- New route: `GET /landing` on the public mux (no auth required)
- `LandingHandler.show()`:
  1. If user is authenticated → redirect to `/`
  2. If host is NOT `atlascloud.app` → redirect to `/login`
  3. Render landing page with brand info
- Update `auth_handler.go`: when redirecting unauthenticated users on `atlascloud.app`, send to `/landing` instead of `/login`
- `atlinks.app` behavior unchanged (still redirects to `/login`)

## New Files

### `internal/handler/landing_handler.go`
- `LandingHandler` struct with `deps *Deps`
- `Register(mux)` adds `GET /landing`
- `show()` with auth check, brand check, render

### `internal/handler/components/pages/landing.templ`
- Standalone `<!DOCTYPE html>` (like login.templ)
- Tailwind CDN with custom theme config inline
- Google Fonts: Manrope (headlines) + Inter (body)
- Material Symbols for icons
- All content from Stitch design with copy fixes
- Links: "Get Started" → `/register`, "Log In" → `/login`

## Modified Files

### `cmd/server/main.go`
- Add `handler.NewLandingHandler(deps).Register(mux)` in `initRoutes()`

### `internal/handler/auth_handler.go`
- Update unauthenticated redirect: `atlascloud.app` → `/landing`, others → `/login`

## CSS Strategy

Standalone Tailwind CDN on the landing page only. The rest of the app continues using `app.css`. The landing page never renders inside the app layout — it's a full standalone HTML page.

## Copy Fixes (from Stitch original)

- Copyright: 2024 → 2025
- "Real-time carrier bidding" → "One-click carrier claims"
- Zone pricing: toned down from "Sophisticated automated rate calculation engine"
- Nav: removed "Solutions" and "Resources" (pages don't exist)
- Kept "No credit card required. Instant activation."

## Images

Stock images from Google Stitch kept as-is for initial implementation. Will be replaced with real screenshots later.
