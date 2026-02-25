# Responsive Design Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add mobile/tablet responsive breakpoints to ATLinks so drivers can use the app on phones and dispatchers can access it remotely without the layout breaking.

**Architecture:** Two breakpoints added to `internal/handler/static/css/app.css` (768px mobile, 1024px tablet). Nav collapses to hamburger menu using Alpine.js at ≤1024px. All changes are CSS-only except for the hamburger toggle markup in `nav.templ`. No template logic changes.

**Tech Stack:** Go templ, Alpine.js (already in stack), vanilla CSS media queries.

---

### Task 1: Add mobile breakpoints to app.css

**Files:**
- Modify: `internal/handler/static/css/app.css`

**Step 1: Add the tablet breakpoint block (≤1024px)**

Append to the bottom of `app.css` (before the final blank line):

```css
/* ============================================================
   Responsive — Tablet (≤1024px)
   ============================================================ */
@media (max-width: 1024px) {
    /* Hide desktop nav menu, show hamburger */
    .nav-menu { display: none; }
    .nav-hamburger { display: flex; }

    /* Notification panel — prevent overflow off right edge */
    .notification-panel { right: 0; width: 300px; }
}
```

**Step 2: Add the mobile breakpoint block (≤768px)**

Append immediately after:

```css
/* ============================================================
   Responsive — Mobile (≤768px)
   ============================================================ */
@media (max-width: 768px) {
    /* Tighten container padding */
    .container { padding: 12px; }

    /* Page header stacks title above buttons */
    .page-header { flex-direction: column; align-items: flex-start; gap: 8px; }

    /* Filter bar stacks to full-width inputs */
    .filter-bar { flex-direction: column; align-items: stretch; }
    .filter-bar .form-control,
    .filter-bar .search-input { min-width: 0 !important; width: 100%; }

    /* Form actions — full-width buttons */
    .form-actions { flex-direction: column; }
    .form-actions .btn { width: 100%; text-align: center; justify-content: center; }

    /* Feedback modal — full width with margin */
    .feedback-modal { width: calc(100vw - 48px); }
    .feedback-overlay { padding: 24px 24px 80px; align-items: flex-end; justify-content: center; }
}
```

**Step 3: Add hamburger base style (hidden by default, shown at tablet)**

Add inside the `/* Navigation */` section, after the `.nav-user` rule:

```css
.nav-hamburger {
    display: none;
    margin-left: auto;
    background: none;
    border: none;
    color: var(--gray-300);
    cursor: pointer;
    padding: 8px;
    font-size: 22px;
    line-height: 1;
}
.nav-hamburger:hover { color: white; }

/* Mobile nav drawer */
.nav-mobile-menu {
    display: none;
    position: absolute;
    top: 48px;
    left: 0;
    right: 0;
    background: var(--gray-800);
    border-top: 1px solid var(--gray-700);
    z-index: 150;
    padding: 8px 0;
    max-height: calc(100vh - 48px);
    overflow-y: auto;
}
.nav-mobile-menu a,
.nav-mobile-menu button {
    display: block;
    width: 100%;
    text-align: left;
    padding: 10px 20px;
    color: var(--gray-300);
    background: none;
    border: none;
    font-size: 14px;
    font-family: inherit;
    cursor: pointer;
    text-decoration: none;
}
.nav-mobile-menu a:hover,
.nav-mobile-menu button:hover { background: var(--gray-700); color: white; text-decoration: none; }
.nav-mobile-menu .mobile-section-label {
    padding: 6px 20px 2px;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    color: var(--gray-500);
    letter-spacing: 0.05em;
}
.nav-mobile-menu hr { border: none; border-top: 1px solid var(--gray-700); margin: 4px 0; }
```

**Step 4: Verify it compiles**

```bash
go build ./...
```
Expected: no errors.

**Step 5: Commit**

```bash
git add internal/handler/static/css/app.css
git commit -m "Add responsive CSS breakpoints and mobile nav styles"
```

---

### Task 2: Add hamburger toggle to nav.templ

**Files:**
- Modify: `internal/handler/components/nav.templ`

**Step 1: Update the nav x-data to include mobile open state**

Change the opening nav tag from:
```html
<nav class="main-nav" x-data="{ open: '' }">
```
To:
```html
<nav class="main-nav" x-data="{ open: '', mobileOpen: false }" @click.outside="mobileOpen = false" style="position:relative">
```

**Step 2: Add hamburger button after `.nav-menu` div**

After the closing `</div>` of `.nav-menu` (line 111 approx), add:

```html
<!-- Hamburger (mobile/tablet) -->
<button class="nav-hamburger" @click="mobileOpen = !mobileOpen" type="button" :aria-expanded="mobileOpen">☰</button>
```

**Step 3: Add mobile nav drawer before closing `</nav>`**

After the `.nav-user` div (before `</nav>`), add:

```html
<!-- Mobile nav drawer -->
<div class="nav-mobile-menu" x-show="mobileOpen" x-cloak>
    <a href="/search/vin">VIN Search</a>
    <hr/>
    <div class="mobile-section-label">Loadboard</div>
    <a href="/loadboard">Browse Loads</a>
    <a href="/loadboard/my-listings">My Listings</a>
    <a href="/loadboard/my-claims">My Claims</a>
    <hr/>
    <div class="mobile-section-label">Dispatch</div>
    <a href="/dispatch/orders">Orders</a>
    <a href="/dispatch/trips">Trips/Loads</a>
    <hr/>
    <div class="mobile-section-label">Accounting</div>
    <a href="/accounting/invoices">Invoices</a>
    <a href="/accounting/payments">Payments</a>
    <a href="/accounting/credit-memos">Credit Memos</a>
    <a href="/accounting/damage-claims">Damage Claims</a>
    <a href="/accounting/ap">Accounts Payable</a>
    <hr/>
    <div class="mobile-section-label">Global</div>
    <a href="/global/customers">Customers</a>
    <a href="/global/employees">Employees</a>
    <a href="/global/trucks">Trucks</a>
    <a href="/global/zones">Zones</a>
    <hr/>
    <div class="mobile-section-label">Reports</div>
    <a href="/reports">All Reports</a>
    <a href="/reports/ar-aging">AR Aging</a>
    <a href="/reports/trip-summary">Trip Summary</a>
    <a href="/reports/driver-settlement">Driver Settlement</a>
    <hr/>
    <div class="mobile-section-label">Utilities</div>
    <a href="/utilities/company">Company Settings</a>
    if user.Role == "super_admin" || user.Role == "company_admin" {
        <a href="/settings/users">Users</a>
    }
    <a href="/feedback">Feedback</a>
    if user.Role == "super_admin" {
        <hr/>
        <div class="mobile-section-label">Admin</div>
        <a href="/admin/companies">Companies</a>
        <a href="/admin/backups">Backups</a>
    }
    <hr/>
    <form method="POST" action="/logout">
        <button type="submit">Logout</button>
    </form>
</div>
```

**Step 4: Regenerate templ output**

```bash
go generate ./...
```
Or if templ is run directly:
```bash
templ generate
```
Expected: `nav_templ.go` updated, no errors.

**Step 5: Build to verify**

```bash
go build ./...
```
Expected: no errors.

**Step 6: Commit**

```bash
git add internal/handler/components/nav.templ internal/handler/components/nav_templ.go
git commit -m "Add hamburger menu for mobile/tablet nav"
```

---

### Task 3: Verify tables have table-container wrappers

**Context:** `.table-container` (which has `overflow-x: auto`) is already present in ~42 template files. This task audits for any tables that were missed.

**Step 1: Find tables NOT inside table-container**

```bash
grep -rn "<table" internal/handler/components/ --include="*.templ" | grep -v "table-container"
```

Review each result. A table missing its wrapper needs `<div class="table-container">...</div>` added around it.

**Step 2: For each unprotected table, wrap it**

Example fix in any `.templ` file:
```html
<!-- Before -->
<table>...</table>

<!-- After -->
<div class="table-container">
    <table>...</table>
</div>
```

**Step 3: Regenerate + build**

```bash
templ generate && go build ./...
```

**Step 4: Commit if any changes were needed**

```bash
git add -p
git commit -m "Wrap remaining tables in table-container for horizontal scroll"
```

---

### Task 4: Manual testing locally

**Step 1: Start the server**

```bash
make run
```
Server runs at `http://localhost:8080`.

**Step 2: Open browser DevTools responsive mode**

In Chrome/Firefox: F12 → toggle device toolbar (Ctrl+Shift+M). Test at these widths:
- 375px (iPhone SE)
- 390px (iPhone 14)
- 768px (iPad)
- 1024px (iPad landscape / small laptop)

**Step 3: Test checklist**

At ≤1024px:
- [ ] Desktop nav menu is hidden
- [ ] Hamburger button (☰) is visible in top-right of nav
- [ ] Tapping hamburger opens mobile drawer with all links
- [ ] Tapping a link closes the drawer and navigates correctly
- [ ] Notification bell still visible in nav

At ≤768px:
- [ ] Page headers stack (title above buttons)
- [ ] Filter bars stack vertically (full-width inputs)
- [ ] Form action buttons are full-width
- [ ] Tables scroll horizontally (not clipped)
- [ ] Feedback modal fits within screen width
- [ ] Dashboard KPI cards wrap to 1-column
- [ ] Login page still centered and usable

**Step 4: Fix any issues found before proceeding**

---

### Task 5: Deploy to production

**Only after Task 4 checklist is fully green.**

```bash
./scripts/deploy.sh
```

Then verify at https://atlinks.app on an actual phone.
