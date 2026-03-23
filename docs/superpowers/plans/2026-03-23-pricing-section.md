# Pricing Section Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a public pricing section to the Atlas Cloud landing page showing Basic/Pro tiers with user-band pricing and an enterprise contact card.

**Architecture:** Single templ component addition to the existing landing page. No backend changes. The pricing section slots between the Feature Highlights and Final CTA sections using the same Tailwind/Stitch design tokens already in the file.

**Tech Stack:** templ (Go HTML templates), Tailwind CSS, Material Symbols

**Spec:** `docs/superpowers/specs/2026-03-23-pricing-section-design.md`
**Mockup:** `.superpowers/brainstorm/1995230-1774306628/pricing-mockup.html`

---

### Task 1: Add pricing section to landing page

**Files:**
- Modify: `internal/handler/components/pages/landing.templ:249-251` (between Feature Highlights and Final CTA)

- [ ] **Step 1: Add the pricing section templ markup**

Insert the following section between the closing `</section>` of the Feature Highlights (after the "Works Everywhere" block, line ~249) and the Final CTA `<section>` (line ~251):

```html
<!-- Pricing Section -->
<section id="pricing" class="py-24 px-6 bg-surface-container-low">
    <div class="max-w-screen-xl mx-auto">
        <div class="text-center mb-16">
            <h2 class="text-4xl font-extrabold tracking-tight text-slate-900 mb-4">Simple, Transparent Pricing</h2>
            <p class="text-secondary max-w-xl mx-auto font-medium">Start small, scale as you grow. No contracts. Cancel anytime.</p>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-5xl mx-auto">
            @pricingCardBasic()
            @pricingCardPro()
            @pricingCardEnterprise()
        </div>
        <!-- Band pricing note -->
        <div class="max-w-3xl mx-auto mt-12 bg-surface-container-lowest rounded-xl border border-outline-variant/20 p-6">
            <h4 class="text-sm font-bold text-slate-900 mb-4 text-center">Growing team? We scale with you.</h4>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-center text-sm">
                <div>
                    <p class="text-on-surface-variant font-medium mb-1">1–3 users</p>
                    <p class="text-slate-900 font-bold">Included</p>
                </div>
                <div>
                    <p class="text-on-surface-variant font-medium mb-1">4–10 users</p>
                    <p class="text-slate-900 font-bold">Basic $69 · Pro $109</p>
                </div>
                <div>
                    <p class="text-on-surface-variant font-medium mb-1">11+ users</p>
                    <p class="text-slate-900 font-bold">+$9/user · +$14/user</p>
                </div>
            </div>
        </div>
    </div>
</section>
```

- [ ] **Step 2: Add the three pricing card templ components**

Add these after the existing `processStep` templ at the bottom of the file (before the closing `}`):

```go
templ pricingCardBasic() {
    <div class="bg-surface-container-lowest rounded-2xl p-8 border border-outline-variant/20 flex flex-col">
        <div class="mb-6">
            <h3 class="text-lg font-bold text-slate-900 mb-1">Basic</h3>
            <p class="text-sm text-secondary">Everything you need to run your operation.</p>
        </div>
        <div class="mb-8">
            <span class="text-5xl font-extrabold text-slate-900">$39</span>
            <span class="text-secondary font-medium">/mo</span>
            <p class="text-xs text-on-surface-variant mt-1">for up to 3 users</p>
        </div>
        <ul class="space-y-3 mb-8 flex-1">
            @pricingFeature("Order Management")
            @pricingFeature("VIN Tracking & NHTSA Decode")
            @pricingFeature("Dispatch & Trip Management")
            @pricingFeature("Invoicing & Payments")
            @pricingFeature("Credit Memos & Damage Claims")
            @pricingFeature("QuickBooks Integration")
            @pricingFeature("10+ Reports & CSV Export")
            @pricingFeature("Dashboard & Analytics")
        </ul>
        <a href="/register" class="block text-center bg-surface-container-high text-slate-900 px-6 py-3.5 rounded-xl font-bold text-sm hover:bg-surface-container transition-all">
            Get Started
        </a>
    </div>
}

templ pricingCardPro() {
    <div class="bg-surface-container-lowest rounded-2xl p-8 border-2 border-primary flex flex-col relative shadow-lg">
        <div class="absolute -top-3.5 left-1/2 -translate-x-1/2 bg-primary text-on-primary text-xs font-bold px-4 py-1 rounded-full tracking-wide uppercase">
            Most Popular
        </div>
        <div class="mb-6">
            <h3 class="text-lg font-bold text-slate-900 mb-1">Pro</h3>
            <p class="text-sm text-secondary">For teams that need the full platform.</p>
        </div>
        <div class="mb-8">
            <span class="text-5xl font-extrabold text-slate-900">$59</span>
            <span class="text-secondary font-medium">/mo</span>
            <p class="text-xs text-on-surface-variant mt-1">for up to 3 users</p>
        </div>
        <ul class="space-y-3 mb-8 flex-1">
            <li class="flex items-start gap-3 text-sm text-on-surface-variant font-medium">
                <span class="material-symbols-outlined text-on-primary-container mt-0.5" style="font-size:18px;">stars</span>
                Everything in Basic, plus:
            </li>
            @pricingFeature("Loadboard Access")
            @pricingFeature("Post & Browse Loads")
            @pricingFeature("Claim Management")
            @pricingFeature("Loadboard Messaging")
        </ul>
        <a href="/register" class="block text-center bg-primary text-on-primary px-6 py-3.5 rounded-xl font-bold text-sm hover:opacity-90 transition-all shadow-lg shadow-black/5">
            Get Started
        </a>
    </div>
}

templ pricingCardEnterprise() {
    <div class="bg-surface-container-lowest rounded-2xl p-8 border border-outline-variant/20 flex flex-col">
        <div class="mb-6">
            <h3 class="text-lg font-bold text-slate-900 mb-1">Need More?</h3>
            <p class="text-sm text-secondary">For larger operations with custom needs.</p>
        </div>
        <div class="mb-8">
            <span class="text-3xl font-extrabold text-slate-900">Let's Talk</span>
        </div>
        <ul class="space-y-3 mb-8 flex-1">
            <li class="flex items-start gap-3 text-sm text-on-surface-variant font-medium">
                <span class="material-symbols-outlined text-on-primary-container mt-0.5" style="font-size:18px;">stars</span>
                Everything in Pro, plus:
            </li>
            @pricingFeature("Custom Integrations")
            @pricingFeature("Volume Pricing")
            @pricingFeature("Priority Support")
            @pricingFeature("Dedicated Onboarding")
        </ul>
        <a href="mailto:support@atlinks.app" class="block text-center bg-surface-container-high text-slate-900 px-6 py-3.5 rounded-xl font-bold text-sm hover:bg-surface-container transition-all">
            Contact Us
        </a>
    </div>
}

templ pricingFeature(name string) {
    <li class="flex items-start gap-3 text-sm text-on-surface-variant">
        <span class="material-symbols-outlined text-on-tertiary-container mt-0.5" style="font-size:18px;">check_circle</span>
        { name }
    </li>
}
```

- [ ] **Step 3: Generate templ output**

Run: `templ generate ./internal/handler/components/pages/`
Expected: generates `landing_templ.go` with no errors

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: compiles clean with no errors

- [ ] **Step 5: Visual verification**

Run the dev server (`make run`) and check `http://localhost:8080` — scroll to the pricing section and verify:
- Three cards render correctly (Basic, Pro highlighted, Need More?)
- "Most Popular" badge on Pro card
- Band pricing bar below cards
- Nav link `#pricing` scrolls to the section
- Mobile responsive (cards stack)

- [ ] **Step 6: Commit**

```bash
git add internal/handler/components/pages/landing.templ internal/handler/components/pages/landing_templ.go
git commit -m "feat: add pricing section to landing page"
```
