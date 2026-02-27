package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/qbo"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

type SyncPaymentArgs struct {
	CompanyID int    `json:"company_id"`
	PaymentID int    `json:"payment_id"`
	Action    string `json:"action"` // "create", "update"
}

func (SyncPaymentArgs) Kind() string { return "qbo_sync_payment" }

type SyncPaymentWorker struct {
	river.WorkerDefaults[SyncPaymentArgs]
	PaymentStore       *store.PaymentStore
	PaymentDetailStore *store.PaymentDetailStore
	InvoiceStore       *store.InvoiceStore
	CustomerStore      *store.CustomerStore
	QBOStore           *store.QBOStore
	RiverClient        *river.Client[pgx.Tx]
	OAuthCfg           *oauth2.Config
	Sandbox            bool
}

func (w *SyncPaymentWorker) Work(ctx context.Context, job *river.Job[SyncPaymentArgs]) error {
	args := job.Args

	conn, err := w.QBOStore.GetConnection(ctx, args.CompanyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get qbo connection: %w", err)
	}

	pmt, err := w.PaymentStore.GetByID(ctx, args.PaymentID)
	if err != nil {
		return fmt.Errorf("load payment %d: %w", args.PaymentID, err)
	}

	if pmt.CustomerID == nil {
		return nil
	}

	cust, err := w.CustomerStore.GetByID(ctx, *pmt.CustomerID)
	if err != nil {
		return fmt.Errorf("load customer: %w", err)
	}
	if cust.QBOCustomerID == nil {
		_, _ = w.RiverClient.Insert(ctx, SyncCustomerArgs{
			CompanyID:  args.CompanyID,
			CustomerID: *pmt.CustomerID,
		}, nil)
		return river.JobSnooze(30 * time.Second)
	}

	details, err := w.PaymentDetailStore.ListByPayment(ctx, args.PaymentID)
	if err != nil {
		return fmt.Errorf("load payment details: %w", err)
	}

	// Build qboInvoiceIDs map: ATLinks invoice_id -> QBO invoice ID
	qboInvoiceIDs := make(map[int]string)
	for _, d := range details {
		if d.InvoiceID == nil {
			continue
		}
		inv, err := w.InvoiceStore.GetByID(ctx, *d.InvoiceID)
		if err != nil {
			continue
		}
		if inv.QBOInvoiceID != nil {
			qboInvoiceIDs[*d.InvoiceID] = *inv.QBOInvoiceID
		}
	}

	client := qbo.NewClient(w.OAuthCfg, w.QBOStore, conn, w.Sandbox)
	qboPmt := qbo.MapPayment(*pmt, details, *cust.QBOCustomerID, qboInvoiceIDs)

	id, syncToken, err := client.UpsertPayment(ctx, qboPmt)
	if err != nil {
		logFail(ctx, w.QBOStore, args.CompanyID, "payment", args.PaymentID, args.Action, err)
		return fmt.Errorf("qbo upsert payment: %w", err)
	}

	_ = w.QBOStore.UpdatePaymentQBO(ctx, args.PaymentID, id, syncToken)
	logOK(ctx, w.QBOStore, args.CompanyID, "payment", args.PaymentID, &id, args.Action)
	return nil
}
