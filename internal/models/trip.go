package models

import "time"

type Trip struct {
	ID               int     `json:"id"`
	CompanyID        int     `json:"company_id"`
	LoadNumber       string  `json:"load_number"`
	Active           bool    `json:"active"`
	TruckNumber      *string `json:"truck_number,omitempty"`
	TruckID          *int    `json:"truck_id,omitempty"`
	TrailerNumber    *string `json:"trailer_number,omitempty"`
	Driver           *string `json:"driver,omitempty"`
	Driver1ID        *int    `json:"driver1_id,omitempty"`
	Driver2          *string `json:"driver2,omitempty"`
	Driver2ID        *int    `json:"driver2_id,omitempty"`
	TripDate         *time.Time `json:"trip_date,omitempty"`
	EstDeliverDate   *time.Time `json:"est_deliver_date,omitempty"`
	DeliverDate      *time.Time `json:"deliver_date,omitempty"`
	ArrivalDate      *time.Time `json:"arrival_date,omitempty"`
	ReturnDate       *time.Time `json:"return_date,omitempty"`
	TotalMileage     *int    `json:"total_mileage,omitempty"`
	TotalFuelGallons *string `json:"total_fuel_gallons,omitempty"`
	FuelAdvance      *string `json:"fuel_advance,omitempty"`
	TripAdvance      *string `json:"trip_advance,omitempty"`
	TollsAdvance     *string `json:"tolls_advance,omitempty"`
	DriverRate       *string `json:"driver_rate,omitempty"`
	DriverCalcType   *string `json:"driver_calc_type,omitempty"`
	DriverAddRate    *string `json:"driver_add_rate,omitempty"`
	DriverAddCalcType *string `json:"driver_add_calc_type,omitempty"`
	TruckRate        *string `json:"truck_rate,omitempty"`
	TruckCalcType    *string `json:"truck_calc_type,omitempty"`
	Comments         *string `json:"comments,omitempty"`
	Status           *string `json:"status,omitempty"`
	EquipmentType    *string `json:"equipment_type,omitempty"`
	Zone             *string `json:"zone,omitempty"`
	Version          int       `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type TripFilter struct {
	Search      string
	Active      string
	DateFrom    string
	DateTo      string
	TruckNumber string
	SortBy      string
	SortDir     string
	Page        int
	PageSize    int
}

type TripListResult struct {
	Items      []Trip
	TotalCount int
	Page       int
	PageSize   int
}

type LoadDetail struct {
	ID            int        `json:"id"`
	CompanyID     int        `json:"company_id"`
	TripID        int        `json:"trip_id"`
	OrderID       *int       `json:"order_id,omitempty"`
	VehicleID     *int       `json:"vehicle_id,omitempty"`
	VIN           *string    `json:"vin,omitempty"`
	Year          *string    `json:"year,omitempty"`
	Make          *string    `json:"make,omitempty"`
	Model         *string    `json:"model,omitempty"`
	Color         *string    `json:"color,omitempty"`
	Weight        *int       `json:"weight,omitempty"`
	Category      *string    `json:"category,omitempty"`
	BayNumber     *string    `json:"bay_number,omitempty"`
	Status        *string    `json:"status,omitempty"`
	LoadedDate    *time.Time `json:"loaded_date,omitempty"`
	DeliveredDate *time.Time `json:"delivered_date,omitempty"`
	Version       int        `json:"version"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
