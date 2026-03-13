package worker

import (
	"context"

	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

func logFail(ctx context.Context, s *store.QBOStore, companyID int, entityType string, entityID int, action string, err error) {
	msg := err.Error()
	_ = s.Log(ctx, &models.QBOSyncLog{
		CompanyID: companyID, EntityType: entityType, EntityID: entityID,
		Action: action, Status: "failed", ErrorMessage: &msg,
	})
}

func logOK(ctx context.Context, s *store.QBOStore, companyID int, entityType string, entityID int, qboID *string, action string) {
	_ = s.Log(ctx, &models.QBOSyncLog{
		CompanyID: companyID, EntityType: entityType, EntityID: entityID,
		QBOID: qboID, Action: action, Status: "success",
	})
}
