# Subscription Management Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Subscription Management page where company admins can view their current plan, see usage metrics, compare tiers, and (eventually) cancel their subscription.

**Architecture:** New templ component under `settings/subscription.templ` rendered by a new handler method on `AdminHandler`. The page displays the company's current subscription (from the existing `SubscriptionStore`), active user count (from `UserStore.ListByCompany`), and static plan comparison data. No new database tables or migrations needed — all data comes from existing stores. The cancel action is a placeholder during beta (no Stripe integration yet).

**Tech Stack:** Go, templ, HTMX, CSS custom properties (Atlas Freight Pro design tokens)

---

## File Structure

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `internal/handler/components/settings/subscription.templ` | Subscription management page UI |
| Modify | `internal/handler/admin_handler.go` | Add subscription page handler + route registration |
| Modify | `internal/handler/components/nav.templ` | Add "Subscription" link to Utilities dropdown |
| Modify | `internal/handler/static/css/app.css` | Add subscription page CSS classes |

---

### Task 1: Add Subscription Page CSS

**Files:**
- Modify: `internal/handler/static/css/app.css` (append to end of file)

- [ ] **Step 1: Add subscription page styles to app.css**

Append these styles to the end of `internal/handler/static/css/app.css`:

```css
/* Subscription Management */
.sub-page { max-width: 960px; margin: 0 auto; }

.sub-plan-card {
    display: flex; gap: 24px; align-items: flex-start;
    background: white; border-radius: var(--radius); padding: 24px;
    box-shadow: var(--shadow); margin-bottom: 24px;
}
.sub-plan-info { flex: 1; }
.sub-plan-price { font-family: var(--font-heading); font-size: 2rem; font-weight: 800; }
.sub-plan-price span { font-size: 0.875rem; font-weight: 400; color: var(--gray-400); }
.sub-plan-meta { display: flex; gap: 24px; margin-top: 12px; color: var(--gray-400); font-size: 0.8125rem; }
.sub-plan-meta dt { font-weight: 600; color: var(--gray-500); }

.sub-badge {
    display: inline-block; padding: 2px 10px; border-radius: 999px;
    font-size: 0.6875rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em;
}
.sub-badge-tier { background: var(--primary); color: white; }
.sub-badge-active { background: #d1fae5; color: #065f46; }
.sub-badge-beta { background: #fef3c7; color: #92400e; }
.sub-badge-suspended { background: #fee2e2; color: #991b1b; }

.sub-metrics { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 24px; }
.sub-metric-card {
    background: white; border-radius: var(--radius); padding: 20px;
    box-shadow: var(--shadow);
}
.sub-metric-label { font-size: 0.75rem; font-weight: 600; color: var(--gray-400); text-transform: uppercase; letter-spacing: 0.05em; }
.sub-metric-value { font-family: var(--font-heading); font-size: 1.5rem; font-weight: 800; margin: 4px 0; }
.sub-metric-value span { font-size: 0.875rem; font-weight: 400; color: var(--gray-400); }
.sub-metric-bar { height: 6px; background: var(--gray-200); border-radius: 3px; margin-top: 8px; overflow: hidden; }
.sub-metric-bar-fill { height: 100%; background: var(--primary); border-radius: 3px; transition: width 0.3s; }

.sub-tiers { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 24px; }
.sub-tier-card {
    background: white; border-radius: var(--radius); padding: 24px;
    box-shadow: var(--shadow); display: flex; flex-direction: column;
}
.sub-tier-card.current { outline: 2px solid var(--primary); }
.sub-tier-name { font-family: var(--font-heading); font-weight: 800; font-size: 1.125rem; }
.sub-tier-price { font-family: var(--font-heading); font-size: 1.75rem; font-weight: 800; margin: 8px 0; }
.sub-tier-price span { font-size: 0.8125rem; font-weight: 400; color: var(--gray-400); }
.sub-tier-features { list-style: none; padding: 0; margin: 16px 0; flex: 1; }
.sub-tier-features li { padding: 4px 0; font-size: 0.8125rem; display: flex; align-items: center; gap: 6px; }
.sub-tier-features .check { color: var(--success); }
.sub-tier-features .lock { color: var(--gray-300); }

.sub-history { background: white; border-radius: var(--radius); padding: 24px; box-shadow: var(--shadow); margin-bottom: 24px; }
.sub-history-empty { text-align: center; padding: 32px; color: var(--gray-400); }

.sub-danger {
    background: #fef2f2; border-radius: var(--radius); padding: 24px; margin-bottom: 24px;
}
.sub-danger h3 { color: var(--danger); margin-bottom: 8px; }
.sub-danger p { color: #991b1b; font-size: 0.8125rem; margin-bottom: 16px; }
.sub-danger-confirm { margin-top: 12px; display: flex; gap: 12px; align-items: flex-end; }
.sub-danger-confirm input { padding: 6px 10px; border: 1px solid var(--gray-300); border-radius: var(--radius-sm); font-size: 0.875rem; }

@media (max-width: 768px) {
    .sub-plan-card { flex-direction: column; }
    .sub-metrics { grid-template-columns: 1fr; }
    .sub-tiers { grid-template-columns: 1fr; }
}
```

- [ ] **Step 2: Verify CSS loads**

Run: `go build ./...`
Expected: Clean compile (CSS is static, just verify no build issues).

- [ ] **Step 3: Commit**

```bash
git add internal/handler/static/css/app.css
git commit -m "feat: add subscription management page CSS"
```

---

### Task 2: Create Subscription Templ Component

**Files:**
- Create: `internal/handler/components/settings/subscription.templ`

- [ ] **Step 1: Create the subscription templ component**

Create `internal/handler/components/settings/subscription.templ`:

```templ
package settings

import (
	"fmt"
	"strings"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

type SubscriptionPageData struct {
	Sub        *models.Subscription
	UserCount  int
	UserLimit  int
	OrderCount int
}

func tierLabel(t models.Tier) string {
	switch t {
	case models.TierPro:
		return "Pro"
	case models.TierEnterprise:
		return "Enterprise"
	default:
		return "Basic"
	}
}

func tierPrice(t models.Tier) string {
	switch t {
	case models.TierPro:
		return "$149"
	case models.TierEnterprise:
		return "$499"
	default:
		return "$49"
	}
}

func tierUserLimit(t models.Tier) int {
	switch t {
	case models.TierEnterprise:
		return 0 // unlimited
	case models.TierPro:
		return 10
	default:
		return 5
	}
}

func userLimitLabel(limit int) string {
	if limit == 0 {
		return "Unlimited"
	}
	return fmt.Sprintf("%d", limit)
}

func usagePercent(used, limit int) int {
	if limit == 0 {
		return 0
	}
	pct := (used * 100) / limit
	if pct > 100 {
		return 100
	}
	return pct
}

templ SubscriptionPage(pg components.PageContext, data SubscriptionPageData) {
	@components.Layout(pg, "Subscription") {
		<div class="sub-page">
			<div class="page-header">
				<h2>Subscription</h2>
				<p style="color:var(--gray-400); margin:0;">Manage your plan, billing details, and enterprise usage.</p>
			</div>

			<!-- Current Plan -->
			@currentPlanCard(data)

			<!-- Usage Metrics -->
			@usageMetrics(data)

			<!-- Plan Comparison -->
			<h3 style="font-family:var(--font-heading); margin-bottom:12px;">Available Plans</h3>
			@planComparison(data)

			<!-- Billing History -->
			@billingHistory()

			<!-- Danger Zone -->
			@dangerZone(pg)
		</div>
	}
}

templ currentPlanCard(data SubscriptionPageData) {
	<div class="sub-plan-card">
		<div class="sub-plan-info">
			<div style="display:flex; align-items:center; gap:8px; margin-bottom:8px;">
				<span class="sub-badge sub-badge-tier">{ tierLabel(data.Sub.Tier) }</span>
				if data.Sub.Status == models.StatusActive {
					<span class="sub-badge sub-badge-active">Active</span>
				} else {
					<span class="sub-badge sub-badge-suspended">Suspended</span>
				}
				<span class="sub-badge sub-badge-beta">Beta</span>
			</div>
			<div class="sub-plan-price">
				{ tierPrice(data.Sub.Tier) }<span>/mo</span>
			</div>
			<dl class="sub-plan-meta">
				<div>
					<dt>Billing</dt>
					<dd>N/A (Beta)</dd>
				</div>
				<div>
					<dt>Next Invoice</dt>
					<dd>N/A (Beta)</dd>
				</div>
				<div>
					<dt>Member Since</dt>
					<dd>{ data.Sub.CreatedAt.Format("Jan 2, 2006") }</dd>
				</div>
			</dl>
		</div>
	</div>
}

templ usageMetrics(data SubscriptionPageData) {
	<div class="sub-metrics">
		<div class="sub-metric-card">
			<div class="sub-metric-label">Active Users</div>
			<div class="sub-metric-value">
				{ fmt.Sprintf("%d", data.UserCount) }
				<span>/ { userLimitLabel(data.UserLimit) }</span>
			</div>
			if data.UserLimit > 0 {
				<div class="sub-metric-bar">
					<div class="sub-metric-bar-fill" style={ fmt.Sprintf("width:%d%%", usagePercent(data.UserCount, data.UserLimit)) }></div>
				</div>
			}
		</div>
		<div class="sub-metric-card">
			<div class="sub-metric-label">Orders This Month</div>
			<div class="sub-metric-value">
				{ fmt.Sprintf("%d", data.OrderCount) }
				<span>/ Unlimited</span>
			</div>
		</div>
		<div class="sub-metric-card">
			<div class="sub-metric-label">Storage Used</div>
			<div class="sub-metric-value">
				—
				<span>/ Unlimited (Beta)</span>
			</div>
		</div>
	</div>
}

type tierInfo struct {
	Tier     models.Tier
	Name     string
	Price    string
	Features []tierFeature
}

type tierFeature struct {
	Name     string
	Included bool
}

func allTiers() []tierInfo {
	return []tierInfo{
		{
			Tier: models.TierBasic, Name: "Basic", Price: "$49",
			Features: []tierFeature{
				{"Dispatch Suite", true},
				{"Accounting", true},
				{"Standard Reports", true},
				{"QuickBooks Sync", true},
				{"Loadboard", false},
				{"EDI Integration", false},
			},
		},
		{
			Tier: models.TierPro, Name: "Pro", Price: "$149",
			Features: []tierFeature{
				{"Full Dispatch Suite", true},
				{"Advanced Accounting", true},
				{"Custom Reports", true},
				{"QuickBooks Sync", true},
				{"Loadboard", true},
				{"EDI Integration", false},
			},
		},
		{
			Tier: models.TierEnterprise, Name: "Enterprise", Price: "$499",
			Features: []tierFeature{
				{"Unlimited Users", true},
				{"EDI Integration", true},
				{"Custom API Access", true},
				{"Priority Support", true},
				{"Loadboard", true},
				{"White-label Portal", true},
			},
		},
	}
}

templ planComparison(data SubscriptionPageData) {
	<div class="sub-tiers">
		for _, t := range allTiers() {
			<div class={ "sub-tier-card", templ.KV("current", t.Tier == data.Sub.Tier) }>
				if t.Tier == data.Sub.Tier {
					<div style="margin-bottom:8px;">
						<span class="sub-badge sub-badge-tier">Current Plan</span>
					</div>
				}
				<div class="sub-tier-name">{ t.Name }</div>
				<div class="sub-tier-price">{ t.Price }<span>/mo</span></div>
				<ul class="sub-tier-features">
					for _, f := range t.Features {
						<li>
							if f.Included {
								<span class="check">&#10003;</span>
							} else {
								<span class="lock">&#128274;</span>
							}
							{ f.Name }
						</li>
					}
				</ul>
				if t.Tier == data.Sub.Tier {
					<button class="btn" disabled>Current Plan</button>
				} else if strings.Compare(t.Tier, data.Sub.Tier) < 0 {
					<button class="btn" disabled title="Coming soon">Downgrade</button>
				} else {
					<button class="btn btn-primary" disabled title="Coming soon">Upgrade</button>
				}
			</div>
		}
	</div>
}

templ billingHistory() {
	<div class="sub-history">
		<h3 style="font-family:var(--font-heading); margin-bottom:16px;">Billing History</h3>
		<table class="data-table" style="margin-bottom:0;">
			<thead>
				<tr>
					<th>Date</th>
					<th>Description</th>
					<th>Amount</th>
					<th>Status</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td colspan="5" class="sub-history-empty">
						No billing history yet &#8212; beta accounts are free
					</td>
				</tr>
			</tbody>
		</table>
	</div>
}

templ dangerZone(pg components.PageContext) {
	<div class="sub-danger">
		<h3>Danger Zone</h3>
		<p>
			Once you cancel your subscription, you will lose access to all premium features at the
			end of your billing cycle. We retain your data for 30 days before permanent deletion.
		</p>
		<div x-data="{ confirm: '' }">
			<label style="font-size:0.8125rem; font-weight:600; color:#991b1b;">
				Type CANCEL to confirm
			</label>
			<div class="sub-danger-confirm">
				<input type="text" x-model="confirm" placeholder="Type CANCEL" />
				<button
					class="btn btn-danger"
					disabled
					title="Not available during beta"
					x-bind:disabled="confirm !== 'CANCEL'"
				>
					Cancel Subscription
				</button>
			</div>
		</div>
	</div>
}
```

- [ ] **Step 2: Generate templ output**

Run: `templ generate ./internal/handler/components/settings/subscription.templ`
Expected: Clean generation, creates `subscription_templ.go`

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Clean compile

- [ ] **Step 4: Commit**

```bash
git add internal/handler/components/settings/subscription.templ internal/handler/components/settings/subscription_templ.go
git commit -m "feat: add subscription management templ component"
```

---

### Task 3: Add Handler and Route

**Files:**
- Modify: `internal/handler/admin_handler.go` — add `showSubscription` method and register route

- [ ] **Step 1: Add the user store interface and field to AdminHandler**

In `internal/handler/admin_handler.go`, add a `subscriptionUserStore` interface near the existing admin interfaces (around line 129):

```go
type subscriptionUserStore interface {
	ListByCompany(ctx context.Context, companyID int) ([]models.User, error)
}
```

Add the field to `AdminHandler` struct (around line 148):

```go
subUserStore subscriptionUserStore
```

Update `NewAdminHandler` to accept and assign it. Add a `subUserStore subscriptionUserStore` parameter and assign it in the return struct: `subUserStore: subUserStore`.

- [ ] **Step 2: Add the showSubscription handler method**

Add this method to `internal/handler/admin_handler.go`:

```go
func (h *AdminHandler) showSubscription(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	user := pg.User
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sub, err := h.subscriptionStore.GetByCompanyID(r.Context(), user.CompanyID)
	if err != nil {
		sub = &models.Subscription{
			CompanyID: user.CompanyID,
			Tier:      models.TierBasic,
			Status:    models.StatusActive,
		}
	}

	users, _ := h.subUserStore.ListByCompany(r.Context(), user.CompanyID)
	activeCount := 0
	for _, u := range users {
		if u.Active {
			activeCount++
		}
	}

	data := settings.SubscriptionPageData{
		Sub:        sub,
		UserCount:  activeCount,
		UserLimit:  settings.TierUserLimitValue(sub.Tier),
		OrderCount: 0, // TODO: wire up order count when needed
	}

	h.deps.renderTempl(w, r, settings.SubscriptionPage(pg, data))
}
```

Note: We need to export `tierUserLimit` as `TierUserLimitValue` from the settings package. Add this to `subscription.templ`:

Actually, since the handler needs to call `tierUserLimit`, we should make it a public function. Update the function in `subscription.templ` from `func tierUserLimit` to `func TierUserLimitValue`.

- [ ] **Step 3: Register the route**

In `AdminHandler.RegisterSettings` (around line 245), add:

```go
mux.Handle("GET /settings/subscription", wrap(h.showSubscription))
```

- [ ] **Step 4: Update NewAdminHandler call in main.go**

In `cmd/server/main.go`, find the `NewAdminHandler(...)` call and add the `userStore` parameter for the new `subUserStore` field. The `userStore` is already created earlier in `main.go` as `routeStores.userStore`, so pass it as an additional argument.

- [ ] **Step 5: Generate and build**

Run: `templ generate ./internal/handler/components/settings/subscription.templ && go build ./...`
Expected: Clean compile

- [ ] **Step 6: Commit**

```bash
git add internal/handler/admin_handler.go internal/handler/components/settings/subscription.templ internal/handler/components/settings/subscription_templ.go cmd/server/main.go
git commit -m "feat: add subscription management handler and route"
```

---

### Task 4: Add Nav Link

**Files:**
- Modify: `internal/handler/components/nav.templ`

- [ ] **Step 1: Add Subscription link to Utilities dropdown**

In `internal/handler/components/nav.templ`, find the Utilities dropdown section (around line 105 where "Company Settings" link is). Add the Subscription link after "Company Settings" and before the Users link:

```templ
<a href="/utilities/company" class="dropdown-item">Company Settings</a>
<a href="/settings/subscription" class="dropdown-item">Subscription</a>
```

This places it right after Company Settings in the Utilities dropdown, visible to all authenticated users.

- [ ] **Step 2: Generate and build**

Run: `templ generate ./internal/handler/components/nav.templ && go build ./...`
Expected: Clean compile

- [ ] **Step 3: Commit**

```bash
git add internal/handler/components/nav.templ internal/handler/components/nav_templ.go
git commit -m "feat: add subscription link to nav utilities dropdown"
```

---

### Task 5: Manual Smoke Test

- [ ] **Step 1: Start the dev server**

Run: `make run` (or `go run ./cmd/server`)

- [ ] **Step 2: Navigate to subscription page**

Open `http://localhost:8080/settings/subscription` in a browser while logged in as a company admin.

Verify:
- Page renders with current plan card showing tier badge, beta badge, and active status
- Usage metrics show active user count and order count
- Plan comparison shows 3 tier cards with current plan highlighted
- Billing history shows empty state message
- Danger zone shows cancel confirmation with CANCEL input
- Nav dropdown under Utilities shows "Subscription" link

- [ ] **Step 3: Final commit (if any tweaks needed)**

```bash
git add -A
git commit -m "fix: subscription page polish"
```
