# Trip Damage View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show all damage records and inspection photos for vehicles on a trip, accessible from the trip detail page; also fix the mobile API to store `trip_id` on damage records so trip-specific damage is tracked correctly.

**Architecture:** Three-part change: (1) mobile API `createDamage` accepts optional `trip_id` and stores it, (2) damage store gains `ListByTrip` that queries by `trip_id` with a vehicle-JOIN fallback for legacy records, (3) trip show page gets a lazy-loaded HTMX "Damage & Inspection Photos" section rendered by a new templ component that groups damage + photos per vehicle.

**Tech Stack:** Go 1.22+, pgx/v5, templ, HTMX, Alpine.js (none new)

---

### Task 1: Fix mobile API — store trip_id on damage records

**Files:**
- Modify: `internal/handler/mobile_handler.go`

**Context:**
`createDamageRequest` currently only sends vehicle-level fields. The `vehicle_damage` table already has a `trip_id` column (nullable). The mobile app knows the trip context and should send it.

**Step 1: Add `trip_id` to the request struct and handler**

In `mobile_handler.go`, update `createDamageRequest` and `createDamage`:

```go
type createDamageRequest struct {
    TripID          *int   `json:"trip_id,omitempty"`
    DamageArea      string `json:"damage_area"`
    DamageType      string `json:"damage_type"`
    DamageSeverity  string `json:"damage_severity"`
    Description     string `json:"description"`
    InspectionPoint string `json:"inspection_point"`
}
```

In `createDamage`, update the `VehicleDamage` construction to include TripID:

```go
d := &models.VehicleDamage{
    VehicleID:       &vehicleID,
    TripID:          req.TripID,   // add this line
    DamageArea:      strPtr(req.DamageArea),
    DamageType:      strPtr(req.DamageType),
    DamageSeverity:  strPtr(req.DamageSeverity),
    Description:     strPtr(req.Description),
    InspectionPoint: strPtr(req.InspectionPoint),
    InspectedBy:     inspectedBy,
    InspectionDate:  &now,
}
```

**Step 2: Build to verify no errors**

```bash
go build ./...
```

Expected: compiles clean with no errors.

**Step 3: Commit**

```bash
git add internal/handler/mobile_handler.go
git commit -m "feat: accept trip_id in mobile damage create endpoint"
```

---

### Task 2: Add ListByTrip to damage store

**Files:**
- Modify: `internal/store/damage_store.go`

**Context:**
`DamageStore` currently only has `ListByVehicle`. We need `ListByTrip` which returns all damage for a trip. Since older records have `trip_id IS NULL`, the query must also pick up records where the vehicle is on the trip (via `load_details` JOIN). Using a single query with an OR condition avoids duplicates.

**Step 1: Add the method to DamageStore**

Add after the existing `ListByVehicle` method:

```go
// ListByTrip returns all damage records for a trip. It returns records directly
// linked by trip_id (new) plus legacy records where the vehicle is on this trip
// (trip_id IS NULL, vehicle linked via load_details).
func (s *DamageStore) ListByTrip(ctx context.Context, tripID int) ([]models.VehicleDamage, error) {
    companyID, err := auth.GetCompanyID(ctx)
    if err != nil {
        return nil, err
    }
    query := fmt.Sprintf(`
        SELECT %s FROM vehicle_damage
        WHERE company_id = $2 AND deleted_at IS NULL
        AND (
            trip_id = $1
            OR (trip_id IS NULL AND vehicle_id IN (
                SELECT vehicle_id FROM load_details
                WHERE trip_id = $1 AND vehicle_id IS NOT NULL AND deleted_at IS NULL
            ))
        )
        ORDER BY vehicle_id, id`, damageColumns)
    rows, err := s.pool.Query(ctx, query, tripID, companyID)
    if err != nil {
        return nil, fmt.Errorf("list damage for trip %d: %w", tripID, err)
    }
    items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.VehicleDamage, error) {
        d, err := scanDamage(row)
        if err != nil {
            return models.VehicleDamage{}, err
        }
        return *d, nil
    })
    if err != nil {
        return nil, fmt.Errorf("scan damage for trip %d: %w", tripID, err)
    }
    return items, nil
}
```

**Step 2: Build to verify**

```bash
go build ./...
```

Expected: clean compile.

**Step 3: Commit**

```bash
git add internal/store/damage_store.go
git commit -m "feat: add DamageStore.ListByTrip with legacy vehicle fallback"
```

---

### Task 3: Create the damage section templ component

**Files:**
- Create: `internal/handler/components/trips/damage_section.templ`

**Context:**
This component receives a slice of per-vehicle groups (each with the load detail, its damage records, and its inspection photos). It renders them as collapsible vehicle cards. Keep it simple — no Alpine needed, just static display with inline thumbnail images.

The component receives a custom struct defined in the handler (passed as a parameter). Define the struct in the same package as the component (`trips` package) so it's accessible without circular imports.

**Step 1: Create the templ file**

```go
package trips

import (
    "fmt"

    "github.com/brady1408/atlinks/internal/handler/components"
    "github.com/brady1408/atlinks/internal/handler/components/attachments"
    "github.com/brady1408/atlinks/internal/models"
    "github.com/brady1408/atlinks/internal/store"
)

// VehicleDamageGroup holds all damage and photos for a single vehicle on a trip.
type VehicleDamageGroup struct {
    Load    store.LoadDetailWithOrder
    Damages []models.VehicleDamage
    Photos  []models.Attachment
}

templ DamageSection(groups []VehicleDamageGroup) {
    if len(groups) == 0 {
        <p class="text-muted">No vehicles on this trip.</p>
    } else {
        for _, g := range groups {
            <div class="card" style="margin-bottom: 1rem;">
                <div class="card-header">
                    <strong>
                        if g.Load.VIN != nil {
                            { *g.Load.VIN }
                        } else {
                            Vehicle { fmt.Sprintf("%d", g.Load.ID) }
                        }
                    </strong>
                    if g.Load.Year != nil || g.Load.Make != nil || g.Load.Model != nil {
                        <span class="text-muted" style="margin-left: 8px; font-size: 0.875rem;">
                            { components.Deref(g.Load.Year) } { components.Deref(g.Load.Make) } { components.Deref(g.Load.Model) }
                        </span>
                    }
                    if g.Load.Color != nil {
                        <span class="text-muted" style="margin-left: 8px; font-size: 0.875rem;">– { components.Deref(g.Load.Color) }</span>
                    }
                </div>
                <div class="card-body">
                    if len(g.Damages) > 0 {
                        <h4 style="margin: 0 0 0.5rem 0; font-size: 0.875rem; text-transform: uppercase; color: var(--text-muted);">Damage Records</h4>
                        <div class="table-container" style="margin-bottom: 1rem;">
                            <table>
                                <thead>
                                    <tr>
                                        <th>Area</th>
                                        <th>Type</th>
                                        <th>Severity</th>
                                        <th>Description</th>
                                        <th>Inspected By</th>
                                        <th>Date</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    for _, d := range g.Damages {
                                        <tr>
                                            <td>{ components.Deref(d.DamageArea) }</td>
                                            <td>{ components.Deref(d.DamageType) }</td>
                                            <td>{ components.Deref(d.DamageSeverity) }</td>
                                            <td>{ components.Deref(d.Description) }</td>
                                            <td>{ components.Deref(d.InspectedBy) }</td>
                                            <td>
                                                if d.InspectionDate != nil {
                                                    { d.InspectionDate.Format("01/02/2006") }
                                                }
                                            </td>
                                        </tr>
                                    }
                                </tbody>
                            </table>
                        </div>
                    } else {
                        <p class="text-muted" style="margin-bottom: 0.75rem;">No damage recorded.</p>
                    }
                    if len(g.Photos) > 0 {
                        <h4 style="margin: 0 0 0.5rem 0; font-size: 0.875rem; text-transform: uppercase; color: var(--text-muted);">Inspection Photos</h4>
                        @attachments.AttachmentList(g.Photos, "vehicle_inspection", components.DerefInt(g.Load.VehicleID))
                    }
                </div>
            </div>
        }
    }
}
```

**Step 2: Generate the templ file**

```bash
go generate ./...
```

Or if no generate directive is set up, run templ directly:

```bash
templ generate
```

Expected: `internal/handler/components/trips/damage_section_templ.go` is created with no errors.

**Step 3: Build**

```bash
go build ./...
```

Expected: clean compile.

**Step 4: Commit**

```bash
git add internal/handler/components/trips/damage_section.templ internal/handler/components/trips/damage_section_templ.go
git commit -m "feat: add DamageSection templ component for trip damage view"
```

---

### Task 4: Wire damage into TripHandler

**Files:**
- Modify: `internal/handler/trip_handler.go`

**Context:**
`TripHandler` already has `attachmentStore` which provides `ListByEntity`. We need to add a `tripDamageStore` interface and field, a new route, and a handler method that builds the `VehicleDamageGroup` slice and renders `DamageSection`.

**Step 1: Add the interface and field**

At the top of `trip_handler.go`, add a new interface after the existing ones:

```go
type tripDamageStore interface {
    ListByTrip(ctx context.Context, tripID int) ([]models.VehicleDamage, error)
}
```

Add `damageStore tripDamageStore` to the `TripHandler` struct:

```go
type TripHandler struct {
    store           tripStore
    loadStore       tripLoadDetailStore
    vehStore        tripVehicleStore
    tripSvc         tripService
    attachmentStore tripAttachmentStore
    damageStore     tripDamageStore     // add this
    deps            *Deps
}
```

Update `NewTripHandler` to accept and store it:

```go
func NewTripHandler(
    store tripStore,
    loadStore tripLoadDetailStore,
    vehStore tripVehicleStore,
    tripSvc tripService,
    attachmentStore tripAttachmentStore,
    damageStore tripDamageStore,          // add this
    deps *Deps,
) *TripHandler {
    return &TripHandler{
        store:           store,
        loadStore:       loadStore,
        vehStore:        vehStore,
        tripSvc:         tripSvc,
        attachmentStore: attachmentStore,
        damageStore:     damageStore,     // add this
        deps:            deps,
    }
}
```

**Step 2: Register the new route**

In `Register`, add after the existing HTMX partials:

```go
mux.HandleFunc("GET /dispatch/trips/{id}/damage", h.damageSection)
```

**Step 3: Add the handler method**

Add at the bottom of the file, before `bindTripForm`:

```go
func (h *TripHandler) damageSection(w http.ResponseWriter, r *http.Request) {
    id, err := parseID(r)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    loads, err := h.loadStore.ListByTripWithOrder(r.Context(), id)
    if err != nil {
        log.Printf("trip damage section: load manifest for trip %d: %v", id, err)
        loads = nil
    }

    damages, err := h.damageStore.ListByTrip(r.Context(), id)
    if err != nil {
        log.Printf("trip damage section: list damage for trip %d: %v", id, err)
        damages = nil
    }

    // Index damage by vehicle_id for grouping.
    damageByVehicle := make(map[int][]models.VehicleDamage)
    for _, d := range damages {
        if d.VehicleID != nil {
            damageByVehicle[*d.VehicleID] = append(damageByVehicle[*d.VehicleID], d)
        }
    }

    // Build one group per vehicle in the load manifest.
    groups := make([]trips.VehicleDamageGroup, 0, len(loads))
    for _, ld := range loads {
        if ld.VehicleID == nil {
            continue
        }
        vehicleID := *ld.VehicleID

        photos, err := h.attachmentStore.ListByEntity(r.Context(), "vehicle_inspection", vehicleID)
        if err != nil {
            log.Printf("trip damage section: list photos for vehicle %d: %v", vehicleID, err)
            photos = nil
        }

        groups = append(groups, trips.VehicleDamageGroup{
            Load:    ld,
            Damages: damageByVehicle[vehicleID],
            Photos:  photos,
        })
    }

    h.deps.renderTempl(w, r, trips.DamageSection(groups))
}
```

Note: you will need to add `"github.com/brady1408/atlinks/internal/handler/components/trips"` to the import block in `trip_handler.go` if it is not already present.

**Step 4: Build**

```bash
go build ./...
```

Expected: compile error in `cmd/server/main.go` because `NewTripHandler` now requires an extra argument — that's expected, fix it in the next task.

**Step 5: Commit (after main.go is fixed in Task 5)**

Hold off — commit together with Task 5.

---

### Task 5: Wire damageStore into main.go

**Files:**
- Modify: `cmd/server/main.go`

**Context:**
`NewTripHandler` now requires a `tripDamageStore`. The existing `damageStore` variable (used by `NewDamageHandler`) satisfies this interface because it has `ListByTrip`. Pass it as the new argument.

**Step 1: Update the NewTripHandler call**

Find the line (around line 311):

```go
handler.NewTripHandler(tripStore, loadDetailStore, vehicleStore, tripSvc, attachmentStore, deps).Register(protectedMux)
```

Change to:

```go
handler.NewTripHandler(tripStore, loadDetailStore, vehicleStore, tripSvc, attachmentStore, damageStore, deps).Register(protectedMux)
```

**Step 2: Build**

```bash
go build ./...
```

Expected: clean compile.

**Step 3: Commit**

```bash
git add internal/handler/trip_handler.go cmd/server/main.go
git commit -m "feat: add trip damage section endpoint to TripHandler"
```

---

### Task 6: Add damage section to trip show page

**Files:**
- Modify: `internal/handler/components/trips/show.templ`

**Context:**
The trip show page already has lazy-loaded sections for Fuel, Expenses, and Routes. We add a "Damage & Inspection Photos" section using the same HTMX `hx-trigger="load"` pattern. Place it after Routes, before Attachments.

**Step 1: Add the section**

In `show.templ`, find the Routes section (around line 187):

```go
<h2 class="section-title">Routes</h2>
<div
    id="route-table"
    hx-get={ fmt.Sprintf("/dispatch/trips/%d/routes", trip.ID) }
    hx-trigger="load"
    hx-swap="innerHTML"
>
    <p class="text-muted">Loading...</p>
</div>
```

Insert the following **after** the Routes section and **before** the Attachments section:

```go
<h2 class="section-title">Damage &amp; Inspection Photos</h2>
<div
    id="damage-section"
    hx-get={ fmt.Sprintf("/dispatch/trips/%d/damage", trip.ID) }
    hx-trigger="load"
    hx-swap="innerHTML"
>
    <p class="text-muted">Loading...</p>
</div>
```

**Step 2: Regenerate templ**

```bash
templ generate
```

Expected: `show_templ.go` updated, no errors.

**Step 3: Build**

```bash
go build ./...
```

Expected: clean compile.

**Step 4: Smoke test**

```bash
make run
```

Navigate to a trip that has vehicles on it. Verify:
- "Damage & Inspection Photos" section appears on the page
- For each vehicle, damage records are shown in a table (or "No damage recorded.")
- Inspection photos (uploaded via mobile) appear as thumbnails with links
- Section loads correctly even for trips with no vehicles

**Step 5: Commit**

```bash
git add internal/handler/components/trips/show.templ internal/handler/components/trips/show_templ.go
git commit -m "feat: add lazy-loaded damage & inspection photos section to trip show page"
```

---

### Task 7: Deploy

```bash
./scripts/deploy.sh
```

Verify on https://atlinks.app that the damage section works on a real trip.
