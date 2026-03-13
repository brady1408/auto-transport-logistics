package connectrpc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/models"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

// --- String pointer helpers ---

func sp(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func ip(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := int32(*v)
	return &i
}

func i32p(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

func timeStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func parseDate(s *string) *time.Time {
	if s == nil {
		return nil
	}
	// Try RFC3339 first, then date-only
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		t, err = time.Parse("2006-01-02", *s)
		if err != nil {
			return nil
		}
	}
	return &t
}

func intToOptStr(v *int) *string {
	if v == nil {
		return nil
	}
	s := fmt.Sprintf("%d", *v)
	return &s
}

func optStrToInt(s *string) *int {
	if s == nil {
		return nil
	}
	n, err := strconv.Atoi(*s)
	if err != nil {
		return nil
	}
	return &n
}

func derefI32(v *int) int32 {
	if v == nil {
		return 0
	}
	return int32(*v)
}

func boolToOptString(b bool) *string {
	if !b {
		return nil
	}
	s := "true"
	return &s
}

func optStringToBool(s *string) bool {
	if s == nil {
		return false
	}
	return *s == "true" || *s == "1" || *s == "yes"
}

// --- Customer converters ---

func customerToProto(c *models.Customer) *pb.Customer {
	return &pb.Customer{
		Id:                int32(c.ID),
		Name:              c.Name,
		Number:            sp(c.Number),
		Address:           sp(c.Address),
		Address2:          sp(c.Address2),
		City:              sp(c.City),
		State:             sp(c.State),
		Zip:               sp(c.Zip),
		Phone:             sp(c.Phone),
		Mobile:            sp(c.Mobile),
		Fax:               sp(c.Fax),
		Contact:           sp(c.Contact),
		Zone:              sp(c.Zone),
		Type:              sp(c.Type),
		Cod:               c.COD,
		Inactive:          c.Inactive,
		CreditLimit:       sp(c.CreditLimit),
		CreditTerms:       sp(c.CreditTerms),
		CombineInvDetLine: c.CombineInvDetLine,
		FuelSurcharge:     sp(c.FuelSurcharge),
		Splc:              sp(c.SPLC),
		RateClass:         sp(c.RateClass),
		RouteCode:         sp(c.RouteCode),
		Comments:          sp(c.Comments),
		DoInstructions:    sp(c.DOInstructions),
		PuInstructions:    sp(c.PUInstructions),
		FuelCalcType:      sp(c.FuelCalcType),
		SalesRep:          sp(c.SalesRep),
		SalesDate:         timeStr(c.SalesDate),
		RevenueClass:      sp(c.RevenueClass),
		Terms:             sp(c.Terms),
		TaxCode:           sp(c.TaxCode),
		LocationType:      sp(c.LocationType),
		Discount:          sp(c.Discount),
		DiscountCalcType:  sp(c.DiscountCalcType),
		CreatedAt:         c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         c.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToCustomerFilter(msg *pb.ListCustomersRequest) models.CustomerFilter {
	f := models.CustomerFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.Search != nil {
		f.Search = *msg.Search
	}
	if msg.Type != nil {
		f.Type = *msg.Type
	}
	if msg.Zone != nil {
		f.Zone = *msg.Zone
	}
	if msg.Active != nil {
		f.Active = *msg.Active
	}
	return f
}

func createCustomerReqToModel(msg *pb.CreateCustomerRequest) *models.Customer {
	return &models.Customer{
		Name:              msg.Name,
		Number:            sp(msg.Number),
		Address:           sp(msg.Address),
		Address2:          sp(msg.Address2),
		City:              sp(msg.City),
		State:             sp(msg.State),
		Zip:               sp(msg.Zip),
		Phone:             sp(msg.Phone),
		Mobile:            sp(msg.Mobile),
		Fax:               sp(msg.Fax),
		Contact:           sp(msg.Contact),
		Zone:              sp(msg.Zone),
		Type:              sp(msg.Type),
		COD:               msg.Cod,
		Inactive:          msg.Inactive,
		CreditLimit:       sp(msg.CreditLimit),
		CreditTerms:       sp(msg.CreditTerms),
		CombineInvDetLine: msg.CombineInvDetLine,
		FuelSurcharge:     sp(msg.FuelSurcharge),
		SPLC:              sp(msg.Splc),
		RateClass:         sp(msg.RateClass),
		RouteCode:         sp(msg.RouteCode),
		Comments:          sp(msg.Comments),
		DOInstructions:    sp(msg.DoInstructions),
		PUInstructions:    sp(msg.PuInstructions),
		FuelCalcType:      sp(msg.FuelCalcType),
		SalesRep:          sp(msg.SalesRep),
		SalesDate:         parseDate(msg.SalesDate),
		RevenueClass:      sp(msg.RevenueClass),
		Terms:             sp(msg.Terms),
		TaxCode:           sp(msg.TaxCode),
		LocationType:      sp(msg.LocationType),
		Discount:          sp(msg.Discount),
		DiscountCalcType:  sp(msg.DiscountCalcType),
	}
}

func updateCustomerReqToModel(msg *pb.UpdateCustomerRequest) *models.Customer {
	c := createCustomerReqToModel(&pb.CreateCustomerRequest{
		Name:              msg.Name,
		Number:            msg.Number,
		Address:           msg.Address,
		Address2:          msg.Address2,
		City:              msg.City,
		State:             msg.State,
		Zip:               msg.Zip,
		Phone:             msg.Phone,
		Mobile:            msg.Mobile,
		Fax:               msg.Fax,
		Contact:           msg.Contact,
		Zone:              msg.Zone,
		Type:              msg.Type,
		Cod:               msg.Cod,
		Inactive:          msg.Inactive,
		CreditLimit:       msg.CreditLimit,
		CreditTerms:       msg.CreditTerms,
		CombineInvDetLine: msg.CombineInvDetLine,
		FuelSurcharge:     msg.FuelSurcharge,
		Splc:              msg.Splc,
		RateClass:         msg.RateClass,
		RouteCode:         msg.RouteCode,
		Comments:          msg.Comments,
		DoInstructions:    msg.DoInstructions,
		PuInstructions:    msg.PuInstructions,
		FuelCalcType:      msg.FuelCalcType,
		SalesRep:          msg.SalesRep,
		SalesDate:         msg.SalesDate,
		RevenueClass:      msg.RevenueClass,
		Terms:             msg.Terms,
		TaxCode:           msg.TaxCode,
		LocationType:      msg.LocationType,
		Discount:          msg.Discount,
		DiscountCalcType:  msg.DiscountCalcType,
	})
	c.ID = int(msg.Id)
	return c
}

// --- Order converters ---

func orderToProto(o *models.Order) *pb.Order {
	return &pb.Order{
		Id:                 int32(o.ID),
		OrderNumber:        o.OrderNumber,
		Active:             o.Active,
		Zone:               sp(o.OriginZone),
		DispatchCode:       sp(o.DispatchCode),
		BolNumber:          sp(o.BOLNumber),
		BillCustomerId:     ip(o.BillCustomerID),
		BillCustomerNumber: sp(o.BillCustomerNumber),
		BillCustomerName:   sp(o.BillCustomerName),
		BillToAddress:      sp(o.BillToAddress),
		BillToAddress2:     sp(o.BillToAddress2),
		BillToCity:         sp(o.BillToCity),
		BillToState:        sp(o.BillToState),
		BillToZip:          sp(o.BillToZip),
		LoadCustomerId:     ip(o.LoadCustomerID),
		LoadCustomerNumber: sp(o.LoadCustomerNumber),
		LoadCustomerName:   sp(o.LoadCustomerName),
		LoadContact:        sp(o.LoadContact),
		LoadPhone:          sp(o.LoadPhone),
		LoadAddress:        sp(o.LoadAddress),
		LoadAddress2:       sp(o.LoadAddress2),
		LoadCity:           sp(o.LoadCity),
		LoadState:          sp(o.LoadState),
		LoadZip:            sp(o.LoadZip),
		DropCustomerId:     ip(o.DropCustomerID),
		DropCustomerNumber: sp(o.DropCustomerNumber),
		DropCustomerName:   sp(o.DropCustomerName),
		DropContact:        sp(o.DropContact),
		DropPhone:          sp(o.DropPhone),
		DropAddress:        sp(o.DropAddress),
		DropAddress2:       sp(o.DropAddress2),
		DropCity:           sp(o.DropCity),
		DropState:          sp(o.DropState),
		DropZip:            sp(o.DropZip),
		ReferenceNumber:    sp(o.ReferenceNumber),
		PoNumber:           sp(o.PONumber),
		SalesRep1:          sp(o.SalesRep1),
		SalesRep2:          sp(o.SalesRep2),
		Comments:           sp(o.Comments),
		PuInstructions:     sp(o.PUInstructions),
		DoInstructions:     sp(o.DOInstructions),
		TransportAmt:       sp(o.TransportAmt),
		TransportCalcType:  sp(o.TransportCalcType),
		FuelSurcharge:      sp(o.FuelSurcharge),
		FuelCalcType:       sp(o.FuelCalcType),
		OtherCharge:        sp(o.OtherCharge),
		Discount:           sp(o.Discount),
		DiscountCalcType:   sp(o.DiscountCalcType),
		TaxRate:            sp(o.TaxRate),
		Tax:                sp(o.Tax),
		TotalCharge:        sp(o.TotalCharge),
		VehicleCount:       int32(o.VehicleCount),
		LoadedCount:        int32(o.LoadedCount),
		DeliveredCount:     int32(o.DeliveredCount),
		ConfirmedCount:     int32(o.ConfirmedCount),
		ScheduledCount:     int32(o.ScheduledCount),
		InvoicedCount:      int32(o.InvoicedCount),
		WaitingCount:       int32(o.WaitingCount),
		StagingCount:       int32(o.StagingCount),
		CreateDate:         timeStr(o.CreateDate),
		EstPickupDate:      timeStr(o.EstPickupDate),
		EstDeliverDate:     timeStr(o.EstDeliverDate),
		EquipmentType:      sp(o.EquipmentType),
		TaxCode:            sp(o.TaxCode),
		DimWeight:          ip(o.DimWeight),
		CreatedAt:          o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          o.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToOrderFilter(msg *pb.ListOrdersRequest) models.OrderFilter {
	f := models.OrderFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.Search != nil {
		f.Search = *msg.Search
	}
	if msg.Zone != nil {
		f.OriginZone = *msg.Zone
	}
	if msg.DispatchCode != nil {
		f.DispatchCode = *msg.DispatchCode
	}
	if msg.Active != nil {
		f.Active = *msg.Active
	}
	if msg.Status != nil {
		f.Status = *msg.Status
	}
	if msg.DateFrom != nil {
		f.DateFrom = *msg.DateFrom
	}
	if msg.DateTo != nil {
		f.DateTo = *msg.DateTo
	}
	return f
}

func createOrderReqToModel(msg *pb.CreateOrderRequest) *models.Order {
	o := &models.Order{
		Active:            msg.Active,
		OriginZone:        sp(msg.Zone),
		DispatchCode:      sp(msg.DispatchCode),
		BOLNumber:         sp(msg.BolNumber),
		BillCustomerID:    i32p(msg.BillCustomerId),
		BillCustomerNumber: sp(msg.BillCustomerNumber),
		BillCustomerName:  sp(msg.BillCustomerName),
		BillToAddress:     sp(msg.BillToAddress),
		BillToAddress2:    sp(msg.BillToAddress2),
		BillToCity:        sp(msg.BillToCity),
		BillToState:       sp(msg.BillToState),
		BillToZip:         sp(msg.BillToZip),
		LoadCustomerID:    i32p(msg.LoadCustomerId),
		LoadCustomerNumber: sp(msg.LoadCustomerNumber),
		LoadCustomerName:  sp(msg.LoadCustomerName),
		LoadContact:       sp(msg.LoadContact),
		LoadPhone:         sp(msg.LoadPhone),
		LoadAddress:       sp(msg.LoadAddress),
		LoadAddress2:      sp(msg.LoadAddress2),
		LoadCity:          sp(msg.LoadCity),
		LoadState:         sp(msg.LoadState),
		LoadZip:           sp(msg.LoadZip),
		DropCustomerID:    i32p(msg.DropCustomerId),
		DropCustomerNumber: sp(msg.DropCustomerNumber),
		DropCustomerName:  sp(msg.DropCustomerName),
		DropContact:       sp(msg.DropContact),
		DropPhone:         sp(msg.DropPhone),
		DropAddress:       sp(msg.DropAddress),
		DropAddress2:      sp(msg.DropAddress2),
		DropCity:          sp(msg.DropCity),
		DropState:         sp(msg.DropState),
		DropZip:           sp(msg.DropZip),
		ReferenceNumber:   sp(msg.ReferenceNumber),
		PONumber:          sp(msg.PoNumber),
		SalesRep1:         sp(msg.SalesRep1),
		SalesRep2:         sp(msg.SalesRep2),
		Comments:          sp(msg.Comments),
		PUInstructions:    sp(msg.PuInstructions),
		DOInstructions:    sp(msg.DoInstructions),
		TransportAmt:      sp(msg.TransportAmt),
		TransportCalcType: sp(msg.TransportCalcType),
		FuelSurcharge:     sp(msg.FuelSurcharge),
		FuelCalcType:      sp(msg.FuelCalcType),
		OtherCharge:       sp(msg.OtherCharge),
		Discount:          sp(msg.Discount),
		DiscountCalcType:  sp(msg.DiscountCalcType),
		TaxRate:           sp(msg.TaxRate),
		Tax:               sp(msg.Tax),
		TotalCharge:       sp(msg.TotalCharge),
		EstPickupDate:     parseDate(msg.EstPickupDate),
		EstDeliverDate:    parseDate(msg.EstDeliverDate),
		EquipmentType:     sp(msg.EquipmentType),
		TaxCode:           sp(msg.TaxCode),
		DimWeight:         i32p(msg.DimWeight),
	}
	if msg.OrderNumber != nil {
		o.OrderNumber = *msg.OrderNumber
	}
	now := time.Now()
	o.CreateDate = &now
	o.OriginalCreateDate = &now
	return o
}

func updateOrderReqToModel(msg *pb.UpdateOrderRequest) *models.Order {
	now := time.Now()
	return &models.Order{
		ID:                 int(msg.Id),
		Active:             msg.Active,
		OriginZone:         sp(msg.Zone),
		DispatchCode:       sp(msg.DispatchCode),
		BOLNumber:          sp(msg.BolNumber),
		BillCustomerID:     i32p(msg.BillCustomerId),
		BillCustomerNumber: sp(msg.BillCustomerNumber),
		BillCustomerName:   sp(msg.BillCustomerName),
		BillToAddress:      sp(msg.BillToAddress),
		BillToAddress2:     sp(msg.BillToAddress2),
		BillToCity:         sp(msg.BillToCity),
		BillToState:        sp(msg.BillToState),
		BillToZip:          sp(msg.BillToZip),
		LoadCustomerID:     i32p(msg.LoadCustomerId),
		LoadCustomerNumber: sp(msg.LoadCustomerNumber),
		LoadCustomerName:   sp(msg.LoadCustomerName),
		LoadContact:        sp(msg.LoadContact),
		LoadPhone:          sp(msg.LoadPhone),
		LoadAddress:        sp(msg.LoadAddress),
		LoadAddress2:       sp(msg.LoadAddress2),
		LoadCity:           sp(msg.LoadCity),
		LoadState:          sp(msg.LoadState),
		LoadZip:            sp(msg.LoadZip),
		DropCustomerID:     i32p(msg.DropCustomerId),
		DropCustomerNumber: sp(msg.DropCustomerNumber),
		DropCustomerName:   sp(msg.DropCustomerName),
		DropContact:        sp(msg.DropContact),
		DropPhone:          sp(msg.DropPhone),
		DropAddress:        sp(msg.DropAddress),
		DropAddress2:       sp(msg.DropAddress2),
		DropCity:           sp(msg.DropCity),
		DropState:          sp(msg.DropState),
		DropZip:            sp(msg.DropZip),
		ReferenceNumber:    sp(msg.ReferenceNumber),
		PONumber:           sp(msg.PoNumber),
		SalesRep1:          sp(msg.SalesRep1),
		SalesRep2:          sp(msg.SalesRep2),
		Comments:           sp(msg.Comments),
		PUInstructions:     sp(msg.PuInstructions),
		DOInstructions:     sp(msg.DoInstructions),
		TransportAmt:       sp(msg.TransportAmt),
		TransportCalcType:  sp(msg.TransportCalcType),
		FuelSurcharge:      sp(msg.FuelSurcharge),
		FuelCalcType:       sp(msg.FuelCalcType),
		OtherCharge:        sp(msg.OtherCharge),
		Discount:           sp(msg.Discount),
		DiscountCalcType:   sp(msg.DiscountCalcType),
		TaxRate:            sp(msg.TaxRate),
		Tax:                sp(msg.Tax),
		TotalCharge:        sp(msg.TotalCharge),
		EditDate:           &now,
		EstPickupDate:      parseDate(msg.EstPickupDate),
		EstDeliverDate:     parseDate(msg.EstDeliverDate),
		EquipmentType:      sp(msg.EquipmentType),
		TaxCode:            sp(msg.TaxCode),
		DimWeight:          i32p(msg.DimWeight),
	}
}

// --- Vehicle converters ---

func vehicleToProto(v *models.OrderVehicle) *pb.Vehicle {
	return &pb.Vehicle{
		Id:                int32(v.ID),
		OrderId:           int32(v.OrderID),
		Active:            v.Active,
		Vin:               sp(v.VIN),
		Year:              sp(v.Year),
		Make:              sp(v.Make),
		Model:             sp(v.Model),
		Color:             sp(v.Color),
		Weight:            ip(v.Weight),
		Category:          sp(v.Category),
		BodyStyle:         sp(v.BodyStyle),
		Status:            v.Status,
		TripId:            ip(v.TripID),
		LoadNumber:        sp(v.LoadNumber),
		BayNumber:         sp(v.BayNumber),
		TransportAmt:      sp(v.TransportAmt),
		TransportCalcType: sp(v.TransportCalcType),
		FuelSurcharge:     sp(v.FuelSurcharge),
		FuelCalcType:      sp(v.FuelCalcType),
		OtherCharge:       sp(v.OtherCharge),
		Discount:          sp(v.Discount),
		DiscountCalcType:  sp(v.DiscountCalcType),
		TaxRate:           sp(v.TaxRate),
		Tax:               sp(v.Tax),
		TotalCharge:       sp(v.TotalCharge),
		ScheduledDate:     timeStr(v.ScheduledDate),
		LoadedDate:        timeStr(v.LoadedDate),
		DeliveredDate:     timeStr(v.DeliveredDate),
		ConfirmedDate:     timeStr(v.ConfirmedDate),
		ConfirmedBy:       sp(v.ConfirmedBy),
		InvoiceNumber:     sp(v.InvoiceNumber),
		InvoiceId:         ip(v.InvoiceID),
		Lot:               sp(v.Lot),
		DamageCode:        sp(v.DamageCode),
		PuDamageCode:      sp(v.PUDamageCode),
		DoDamageCode:      sp(v.DODamageCode),
		Comments:          sp(v.Comments),
		RateClass:         sp(v.RateClass),
		DimLength:         sp(v.DimLength),
		DimWidth:          sp(v.DimWidth),
		DimHeight:         sp(v.DimHeight),
		RunDrive:          v.RunDrive,
		Operable:          v.Operable,
		CreatedAt:         v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         v.UpdatedAt.Format(time.RFC3339),
	}
}

func createVehicleReqToModel(msg *pb.CreateVehicleRequest) *models.OrderVehicle {
	return &models.OrderVehicle{
		OrderID:           int(msg.OrderId),
		Active:            msg.Active,
		VIN:               sp(msg.Vin),
		Year:              sp(msg.Year),
		Make:              sp(msg.Make),
		Model:             sp(msg.Model),
		Color:             sp(msg.Color),
		Weight:            i32p(msg.Weight),
		Category:          sp(msg.Category),
		BodyStyle:         sp(msg.BodyStyle),
		Status:            "Waiting",
		TransportAmt:      sp(msg.TransportAmt),
		TransportCalcType: sp(msg.TransportCalcType),
		FuelSurcharge:     sp(msg.FuelSurcharge),
		FuelCalcType:      sp(msg.FuelCalcType),
		OtherCharge:       sp(msg.OtherCharge),
		Discount:          sp(msg.Discount),
		DiscountCalcType:  sp(msg.DiscountCalcType),
		TaxRate:           sp(msg.TaxRate),
		Tax:               sp(msg.Tax),
		TotalCharge:       sp(msg.TotalCharge),
		Lot:               sp(msg.Lot),
		Comments:          sp(msg.Comments),
		RateClass:         sp(msg.RateClass),
		DimLength:         sp(msg.DimLength),
		DimWidth:          sp(msg.DimWidth),
		DimHeight:         sp(msg.DimHeight),
		RunDrive:          msg.RunDrive,
		Operable:          msg.Operable,
	}
}

func updateVehicleReqToModel(msg *pb.UpdateVehicleRequest) *models.OrderVehicle {
	return &models.OrderVehicle{
		ID:                int(msg.Id),
		Active:            msg.Active,
		VIN:               sp(msg.Vin),
		Year:              sp(msg.Year),
		Make:              sp(msg.Make),
		Model:             sp(msg.Model),
		Color:             sp(msg.Color),
		Weight:            i32p(msg.Weight),
		Category:          sp(msg.Category),
		BodyStyle:         sp(msg.BodyStyle),
		TransportAmt:      sp(msg.TransportAmt),
		TransportCalcType: sp(msg.TransportCalcType),
		FuelSurcharge:     sp(msg.FuelSurcharge),
		FuelCalcType:      sp(msg.FuelCalcType),
		OtherCharge:       sp(msg.OtherCharge),
		Discount:          sp(msg.Discount),
		DiscountCalcType:  sp(msg.DiscountCalcType),
		TaxRate:           sp(msg.TaxRate),
		Tax:               sp(msg.Tax),
		TotalCharge:       sp(msg.TotalCharge),
		Lot:               sp(msg.Lot),
		Comments:          sp(msg.Comments),
		RateClass:         sp(msg.RateClass),
		DimLength:         sp(msg.DimLength),
		DimWidth:          sp(msg.DimWidth),
		DimHeight:         sp(msg.DimHeight),
		RunDrive:          msg.RunDrive,
		Operable:          msg.Operable,
	}
}
