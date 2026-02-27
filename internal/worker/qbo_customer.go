package worker

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/brady1408/atlinks/internal/qbo"
	"github.com/brady1408/atlinks/internal/riverargs"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

// SyncCustomerArgs is re-exported from riverargs for backward compatibility.
type SyncCustomerArgs = riverargs.SyncCustomerArgs

type SyncCustomerWorker struct {
	river.WorkerDefaults[SyncCustomerArgs]
	CustomerStore *store.CustomerStore
	QBOStore      *store.QBOStore
	OAuthCfg      *oauth2.Config
	Sandbox       bool
}

func (w *SyncCustomerWorker) Work(ctx context.Context, job *river.Job[SyncCustomerArgs]) error {
	args := job.Args

	conn, err := w.QBOStore.GetConnection(ctx, args.CompanyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get qbo connection: %w", err)
	}

	cust, err := w.CustomerStore.GetByID(ctx, args.CustomerID)
	if err != nil {
		return fmt.Errorf("load customer %d: %w", args.CustomerID, err)
	}

	client := qbo.NewClient(w.OAuthCfg, w.QBOStore, conn, w.Sandbox)
	qboCust := qbo.MapCustomer(*cust)

	action := "create"
	if cust.QBOCustomerID != nil {
		action = "update"
	}

	qboID, err := client.UpsertCustomer(ctx, qboCust)
	if err != nil {
		logFail(ctx, w.QBOStore, args.CompanyID, "customer", args.CustomerID, action, err)
		return fmt.Errorf("qbo upsert customer: %w", err)
	}

	if err := w.QBOStore.UpdateCustomerQBOID(ctx, args.CustomerID, qboID); err != nil {
		log.Printf("warn: update customer qbo_id: %v", err)
	}

	logOK(ctx, w.QBOStore, args.CompanyID, "customer", args.CustomerID, &qboID, action)
	return nil
}
