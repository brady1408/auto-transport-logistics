package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/qbo"
	"github.com/brady1408/atlinks/internal/riverargs"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

// SyncInvoiceArgs is re-exported from riverargs for backward compatibility.
type SyncInvoiceArgs = riverargs.SyncInvoiceArgs

type SyncInvoiceWorker struct {
	river.WorkerDefaults[SyncInvoiceArgs]
	InvoiceStore       *store.InvoiceStore
	InvoiceDetailStore *store.InvoiceDetailStore
	CustomerStore      *store.CustomerStore
	QBOStore           *store.QBOStore
	RiverClient        *river.Client[pgx.Tx]
	OAuthCfg           *oauth2.Config
	Sandbox            bool
}

func (w *SyncInvoiceWorker) Work(ctx context.Context, job *river.Job[SyncInvoiceArgs]) error {
	args := job.Args

	conn, err := w.QBOStore.GetConnection(ctx, args.CompanyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get qbo connection: %w", err)
	}

	inv, err := w.InvoiceStore.GetByID(ctx, args.InvoiceID)
	if err != nil {
		return fmt.Errorf("load invoice %d: %w", args.InvoiceID, err)
	}

	if inv.CustomerID == nil {
		return nil
	}

	cust, err := w.CustomerStore.GetByID(ctx, *inv.CustomerID)
	if err != nil {
		return fmt.Errorf("load customer: %w", err)
	}
	if cust.QBOCustomerID == nil {
		_, _ = w.RiverClient.Insert(ctx, SyncCustomerArgs{
			CompanyID:  args.CompanyID,
			CustomerID: *inv.CustomerID,
		}, nil)
		return river.JobSnooze(30 * time.Second)
	}

	client := qbo.NewClient(w.OAuthCfg, w.QBOStore, conn, w.Sandbox)

	if args.Action == "void" && inv.QBOInvoiceID != nil {
		syncToken := ""
		if inv.QBOSyncToken != nil {
			syncToken = *inv.QBOSyncToken
		}
		if err := client.VoidInvoice(ctx, *inv.QBOInvoiceID, syncToken); err != nil {
			logFail(ctx, w.QBOStore, args.CompanyID, "invoice", args.InvoiceID, "void", err)
			return err
		}
		logOK(ctx, w.QBOStore, args.CompanyID, "invoice", args.InvoiceID, inv.QBOInvoiceID, "void")
		return nil
	}

	details, err := w.InvoiceDetailStore.ListByInvoice(ctx, args.InvoiceID)
	if err != nil {
		return fmt.Errorf("load invoice details: %w", err)
	}

	qboInv := qbo.MapInvoice(*inv, details, *cust.QBOCustomerID)

	id, syncToken, err := client.UpsertInvoice(ctx, qboInv)
	if err != nil {
		var staleErr *qbo.SyncTokenError
		if errors.As(err, &staleErr) && inv.QBOInvoiceID != nil {
			freshToken, fetchErr := client.GetInvoiceSyncToken(ctx, *inv.QBOInvoiceID)
			if fetchErr == nil {
				qboInv.SyncToken = freshToken
				id, syncToken, err = client.UpsertInvoice(ctx, qboInv)
			}
		}
		if err != nil {
			logFail(ctx, w.QBOStore, args.CompanyID, "invoice", args.InvoiceID, args.Action, err)
			return fmt.Errorf("qbo upsert invoice: %w", err)
		}
	}

	_ = w.QBOStore.UpdateInvoiceQBO(ctx, args.InvoiceID, id, syncToken)
	logOK(ctx, w.QBOStore, args.CompanyID, "invoice", args.InvoiceID, &id, args.Action)
	return nil
}
