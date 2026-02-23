package models

import "time"

type OrderCharge struct {
	ID          int     `json:"id"`
	OrderID     *int    `json:"order_id,omitempty"`
	VehicleID   *int    `json:"vehicle_id,omitempty"`
	TripID      *int    `json:"trip_id,omitempty"`
	Description *string `json:"description,omitempty"`
	Amount      *string `json:"amount,omitempty"`
	ItemCode    *string `json:"item_code,omitempty"`
	Qty         *int    `json:"qty,omitempty"`
	Rate        *string `json:"rate,omitempty"`
	CalcType    *string `json:"calc_type,omitempty"`
	Taxable     bool    `json:"taxable"`
	Billable    bool    `json:"billable"`
	APPayable   bool    `json:"ap_payable"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
