# Damage-Linked Photo Upload Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Link inspection photos to specific damage records so the trip damage view can show photos inline under the damage record they document.

**Architecture:** New mobile API endpoint `POST /api/v1/driver/vehicles/{vehicleID}/damage/{damageID}/photos` stores attachments with `category = "vehicle_damage"` and `entity_id = damageID`. The trip damage view handler fetches photos per damage record and passes them to the component via a new `DamageWithPhotos` struct. The templ component is restructured to show each damage record as a card with its photos below.

**Tech Stack:** Go 1.22+, pgx/v5, templ, HTMX (no new dependencies)

---

### Task 1: Add damage photo upload endpoint to mobile API

**Files:**
- Modify: `internal/handler/mobile_handler.go`

**Context:**
`uploadPhoto` currently only handles `POST /api/v1/driver/vehicles/{id}/photos` (vehicle-level photos, `category = "vehicle_inspection"`). We need a parallel endpoint for damage-specific photos. The new endpoint parses both `vehicleID` and `damageID` from the URL; everything else is identical except `category = "vehicle_damage"` and `entity_id = damageID`.

**Step 1: Register the new route**

In `Register`, add after the existing photos route:

```go
mux.HandleFunc("POST /api/v1/driver/vehicles/{vehicleID}/damage/{damageID}/photos", h.uploadDamagePhoto)
```

Note: the existing route uses `{id}` for vehicleID. The new route uses named segments `{vehicleID}` and `{damageID}` to distinguish them.

**Step 2: Add the handler method**

Add after the existing `uploadPhoto` method:

```go
func (h *MobileHandler) uploadDamagePhoto(w http.ResponseWriter, r *http.Request) {
    damageID, err := parsePathID(r, "damageID")
    if err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid damage ID")
        return
    }

    user, ok := auth.GetUserFromRequest(r)
    if !ok {
        h.writeError(w, http.StatusUnauthorized, "unauthorized")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 25<<20)

    file, header, err := r.FormFile("file")
    if err != nil {
        var maxBytesErr *http.MaxBytesError
        if errors.As(err, &maxBytesErr) {
            h.writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 25MB)")
            return
        }
        h.writeError(w, http.StatusBadRequest, "no file provided")
        return
    }
    defer file.Close()

    buf := make([]byte, 512)
    n, _ := file.Read(buf)
    contentType := http.DetectContentType(buf[:n])
    if _, err := file.Seek(0, io.SeekStart); err != nil {
        h.writeError(w, http.StatusInternalServerError, "failed to process file")
        return
    }

    if !mobileAllowedImageTypes[contentType] {
        h.writeError(w, http.StatusBadRequest, "only image files allowed (JPEG, PNG, GIF, WebP)")
        return
    }

    ext := filepath.Ext(header.Filename)
    if ext == "" {
        exts, _ := mime.ExtensionsByType(contentType)
        if len(exts) > 0 {
            ext = exts[0]
        }
    }

    storageKey, written, err := h.storageSvc.Save(user.CompanyID, "vehicle_damage", damageID, ext, file)
    if err != nil {
        log.Printf("mobile api: save damage photo: %v", err)
        h.writeError(w, http.StatusInternalServerError, "failed to save file")
        return
    }

    att := &models.Attachment{
        CompanyID:   user.CompanyID,
        Category:    "vehicle_damage",
        EntityID:    damageID,
        Filename:    header.Filename,
        StorageKey:  storageKey,
        ContentType: contentType,
        SizeBytes:   written,
        UploadedBy:  &user.ID,
    }

    if err := h.attachmentStore.Create(r.Context(), att); err != nil {
        h.storageSvc.Delete(storageKey)
        log.Printf("mobile api: create damage attachment record: %v", err)
        h.writeError(w, http.StatusInternalServerError, "failed to save attachment")
        return
    }

    h.deps.Audit.Log(r.Context(), "attachments", att.ID, "INSERT", nil, att)

    h.writeJSON(w, http.StatusCreated, map[string]any{
        "id":           att.ID,
        "filename":     att.Filename,
        "content_type": att.ContentType,
        "size_bytes":   att.SizeBytes,
    })
}
```

**Step 3: Build**

```bash
go build ./...
```

Expected: clean compile. If `parsePathID` is not defined, check `internal/handler/helpers.go` or similar — it is used by `trip_handler.go` for `loadID`, so it already exists.

**Step 4: Commit**

```bash
git add internal/handler/mobile_handler.go
git commit -m "feat: add damage-specific photo upload endpoint to mobile API"
```

---

### Task 2: Restructure the damage component to show photos per damage record

**Files:**
- Modify: `internal/handler/components/trips/damage_section.templ`

**Context:**
Currently `VehicleDamageGroup` has a flat `Damages []models.VehicleDamage` and `Photos []models.Attachment` (vehicle-level photos). We need to add a `DamageWithPhotos` struct that pairs each damage record with its specific photos, and update the component to render them as individual cards rather than a table.

Keep the existing `Photos []models.Attachment` field on `VehicleDamageGroup` for general vehicle inspection photos (unlinked to damage) so those continue to render as before.

**Step 1: Update the structs and component**

Replace the entire file content:

```go
package trips

import (
	"fmt"

	"github.com/brady1408/atlinks/internal/handler/components"
	"github.com/brady1408/atlinks/internal/handler/components/attachments"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

// DamageWithPhotos pairs a damage record with photos taken specifically for it.
type DamageWithPhotos struct {
	Damage models.VehicleDamage
	Photos []models.Attachment
}

// VehicleDamageGroup holds all damage (with their photos) and general inspection photos for one vehicle.
type VehicleDamageGroup struct {
	Load             store.LoadDetailWithOrder
	DamagesWithPhotos []DamageWithPhotos
	InspectionPhotos  []models.Attachment
}

templ DamageSection(groups []VehicleDamageGroup) {
	if len(groups) == 0 {
		<p class="text-muted">No damage or inspection photos recorded for this trip.</p>
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
					if len(g.DamagesWithPhotos) > 0 {
						<h4 style="margin: 0 0 0.75rem 0; font-size: 0.875rem; text-transform: uppercase; color: var(--text-muted);">Damage Records</h4>
						for _, dp := range g.DamagesWithPhotos {
							<div style="border: 1px solid var(--border-color); border-radius: 4px; padding: 0.75rem; margin-bottom: 0.75rem;">
								<div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem; margin-bottom: 0.5rem;">
									<div>
										<span class="detail-label">Area</span>
										<div>{ components.Deref(dp.Damage.DamageArea) }</div>
									</div>
									<div>
										<span class="detail-label">Type</span>
										<div>{ components.Deref(dp.Damage.DamageType) }</div>
									</div>
									<div>
										<span class="detail-label">Severity</span>
										<div>{ components.Deref(dp.Damage.DamageSeverity) }</div>
									</div>
								</div>
								if dp.Damage.Description != nil {
									<div style="margin-bottom: 0.5rem;">
										<span class="detail-label">Description</span>
										<div>{ components.Deref(dp.Damage.Description) }</div>
									</div>
								}
								<div style="display: flex; gap: 1.5rem; font-size: 0.875rem; color: var(--text-muted);">
									if dp.Damage.InspectedBy != nil {
										<span>Inspected by { components.Deref(dp.Damage.InspectedBy) }</span>
									}
									if dp.Damage.InspectionDate != nil {
										<span>{ dp.Damage.InspectionDate.Format("01/02/2006") }</span>
									}
								</div>
								if len(dp.Photos) > 0 {
									<div style="margin-top: 0.75rem;">
										@attachments.AttachmentList(dp.Photos, "vehicle_damage", dp.Damage.ID)
									</div>
								}
							</div>
						}
					}
					if len(g.InspectionPhotos) > 0 {
						<h4 style="margin: 0 0 0.5rem 0; font-size: 0.875rem; text-transform: uppercase; color: var(--text-muted);">Inspection Photos</h4>
						@attachments.AttachmentList(g.InspectionPhotos, "vehicle_inspection", components.DerefInt(g.Load.VehicleID))
					}
				</div>
			</div>
		}
	}
}
```

**Step 2: Regenerate**

```bash
templ generate
```

Expected: `damage_section_templ.go` regenerated, no errors.

**Step 3: Build — expect compile errors in trip_handler.go**

```bash
go build ./...
```

Expected: compile errors because `trip_handler.go` still references the old `VehicleDamageGroup` fields (`Damages`, `Photos`). Fix in Task 3.

---

### Task 3: Update trip handler to build DamageWithPhotos

**Files:**
- Modify: `internal/handler/trip_handler.go`

**Context:**
The `damageSection` handler must now fetch `vehicle_damage` photos per damage record (not just vehicle-level inspection photos) and build the new `DamageWithPhotos` slice.

**Step 1: Update the group-building loop in `damageSection`**

Find the block starting with `// Build one group per vehicle in the load manifest.` and replace it entirely:

```go
// Build one group per vehicle in the load manifest.
groups := make([]trips.VehicleDamageGroup, 0, len(loads))
for _, ld := range loads {
    if ld.VehicleID == nil {
        continue
    }
    vehicleID := *ld.VehicleID

    // Fetch damage-linked photos for each damage record on this vehicle.
    vdamages := damageByVehicle[vehicleID]
    damagesWithPhotos := make([]trips.DamageWithPhotos, 0, len(vdamages))
    for _, d := range vdamages {
        photos, err := h.attachmentStore.ListByEntity(r.Context(), "vehicle_damage", d.ID)
        if err != nil {
            log.Printf("trip damage section: list damage photos for damage %d: %v", d.ID, err)
            photos = nil
        }
        damagesWithPhotos = append(damagesWithPhotos, trips.DamageWithPhotos{
            Damage: d,
            Photos: photos,
        })
    }

    // General vehicle inspection photos (not linked to a specific damage record).
    inspectionPhotos, err := h.attachmentStore.ListByEntity(r.Context(), "vehicle_inspection", vehicleID)
    if err != nil {
        log.Printf("trip damage section: list inspection photos for vehicle %d: %v", vehicleID, err)
        inspectionPhotos = nil
    }

    // Skip vehicles with nothing to show.
    if len(damagesWithPhotos) == 0 && len(inspectionPhotos) == 0 {
        continue
    }

    groups = append(groups, trips.VehicleDamageGroup{
        Load:              ld,
        DamagesWithPhotos: damagesWithPhotos,
        InspectionPhotos:  inspectionPhotos,
    })
}
```

**Step 2: Remove the now-redundant `vdamages` local variable**

The earlier quick-fix code set `vdamages := damageByVehicle[vehicleID]` and checked `len(vdamages) == 0`. That logic is now inside the new loop above. Remove the old `vdamages` assignment and the old `continue` check if they remain — the new loop handles it.

**Step 3: Build**

```bash
go build ./...
```

Expected: clean compile.

**Step 4: Regenerate templ (in case of any template changes)**

```bash
templ generate
```

**Step 5: Smoke test**

```bash
make run
```

Navigate to a trip with damage records. Verify:
- Each damage record renders as its own card with area/type/severity/description fields
- Photos uploaded via `POST /api/v1/driver/vehicles/{vehicleID}/damage/{damageID}/photos` appear under the correct damage card
- General vehicle inspection photos still appear in the "Inspection Photos" section
- Vehicles with no damage and no photos are not shown
- Empty trip shows "No damage or inspection photos recorded for this trip."

**Step 6: Commit**

```bash
git add internal/handler/trip_handler.go internal/handler/components/trips/damage_section.templ internal/handler/components/trips/damage_section_templ.go
git commit -m "feat: show damage-specific photos inline under each damage record on trip view"
```

---

### Task 4: Deploy

```bash
./scripts/deploy.sh
```

Verify on https://atlinks.app that damage photos uploaded via the mobile app's new endpoint appear under the correct damage record on the trip page.
