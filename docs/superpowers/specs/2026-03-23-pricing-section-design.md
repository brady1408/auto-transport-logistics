# Pricing Section Design

**Date:** 2026-03-23
**Status:** Approved

## Overview

Add a public pricing section to the Atlas Cloud landing page at `#pricing`. Displays two tiers (Basic, Pro) with user-band pricing, plus a "Need More?" contact card for enterprise inquiries.

## Pricing Model

**Model: Value Anchor** — budget-friendly entry with a tight Basic→Pro gap to encourage upsell.

**Billing:** Monthly only (no annual option for now).

### Price Table

| Band | Basic | Pro |
|------|-------|-----|
| 1–3 users | $39/mo | $59/mo |
| 4–10 users | $69/mo | $109/mo |
| 11+ users | $69 + $9/user/mo | $109 + $14/user/mo |

### Feature Breakdown

**Basic ($39/mo, 1–3 users):**
- Order Management
- VIN Tracking & NHTSA Decode
- Dispatch & Trip Management
- Invoicing & Payments
- Credit Memos & Damage Claims
- QuickBooks Integration
- 10+ Reports & CSV Export
- Dashboard & Analytics

**Pro ($59/mo, 1–3 users):**
- Everything in Basic
- Loadboard Access
- Post & Browse Loads
- Claim Management
- Loadboard Messaging

**Need More? (Contact Us):**
- Everything in Pro
- Custom Integrations
- Volume Pricing
- Priority Support
- Dedicated Onboarding

## Layout

The pricing section sits between the "Feature Highlights" section and the "Final CTA" section on the landing page.

### Structure

Three equal-width cards in a responsive grid (stacks on mobile):

1. **Basic card** — white background, subtle border, grey "Get Started" button
2. **Pro card (featured)** — white background, dark border + shadow, "Most Popular" badge at top, primary-color "Get Started" button
3. **Need More? card** — white background, subtle border, "Let's Talk" heading instead of a price, grey "Contact Us" button linking to email

Below the cards, a compact band-pricing bar shows scaling rates:
- "Growing team? We scale with you."
- Three columns: 1–3 users (Included), 4–10 users (Basic $69 · Pro $109), 11+ users (+$9/user · +$14/user)

### Visual Design

- Matches existing Stitch theme: Manrope headings, Inter body, project color tokens
- Pro card uses `border-2 border-primary` + `shadow-lg` + absolute-positioned badge
- Basic and Need More? cards use `border border-outline-variant/20`
- Green check_circle icons for feature list items
- Blue stars icon for "Everything in [tier], plus:" lines
- Band pricing bar is a rounded card with `outline-variant/20` border

## Technical Scope

### Files to Modify

1. **`internal/handler/components/pages/landing.templ`** — Add `#pricing` section between Feature Highlights and Final CTA. New `templ` for the pricing section inlined in this file (no separate component needed — it's landing-page-specific).

### Files NOT Modified

- No backend changes — pricing is purely presentational on the landing page
- No subscription model changes — tiers/features already exist in `internal/models/subscription.go`
- No registration flow changes — users still default to Basic on signup
- No billing integration — that's a separate future project

### Implementation Notes

- The pricing section uses the same Tailwind classes and color tokens already present in the landing page
- Responsive: 3-column grid on desktop (`md:grid-cols-3`), stacks to single column on mobile
- Follow existing landing page border-radius patterns (`rounded-2xl` for cards, `rounded-xl` for inner elements)
- "Get Started" buttons link to `/register`
- "Contact Us" button links to `mailto:support@atlinks.app`
- The `#pricing` anchor already exists in the nav — section just needs the matching `id="pricing"`
- Feature lists on the cards are **marketing copy**, not a 1:1 mapping to `Feature` constants in `subscription.go`. Items like "Dashboard & Analytics" are available to all tiers — they appear on Basic to show value, not because they're gated.
- "Most Popular" badge on Pro card: `bg-primary text-on-primary text-xs font-bold px-4 py-1 rounded-full tracking-wide uppercase`, absolutely positioned centered at the top edge of the card with `-top-3.5` and `-translate-x-1/2`
- Card internal layout order (top to bottom): badge (Pro only) → tier name + subtitle → price + billing period + user count → feature checklist → CTA button
- "Everything in [tier], plus:" line uses `stars` Material Symbol in `text-on-primary-container`; feature items use `check_circle` in `text-on-tertiary-container`
- Band pricing bar sits inside the same `max-w-screen-xl` container, uses a 3-column grid that stacks on mobile

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Pricing model | Banded with per-user tail | Captures value from growing teams without punishing small operators |
| Entry price | $39/mo (Basic) | Budget-friendly, land-and-expand strategy |
| Basic→Pro gap | $20 at each band | Tight enough to make Pro feel like a no-brainer |
| Enterprise card | "Need More?" with Contact Us | Keeps page focused on Basic/Pro; no EDI mention (not ready) |
| Annual billing | Not yet | Monthly only for simplicity; can add toggle later |
| Plan selection at signup | No change | Still defaults to Basic; self-serve upgrade is a separate project |
| User limit enforcement | Not in scope | Display only — actual enforcement is a separate project |

## Out of Scope

- Billing/payment integration (Stripe, etc.)
- Self-serve plan upgrade flow
- User count enforcement in the app
- Annual billing toggle
- Trial periods
- EDI/Enterprise tier details
