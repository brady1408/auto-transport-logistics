package connectrpc

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/brady1408/atlinks/internal/audit"
	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
	"github.com/brady1408/atlinks/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/atlinks/internal/models"
)

type employeeStore interface {
	List(ctx context.Context, f models.EmployeeFilter) (*models.EmployeeListResult, error)
	GetByID(ctx context.Context, id int) (*models.Employee, error)
	Create(ctx context.Context, e *models.Employee) error
	Update(ctx context.Context, e *models.Employee) error
	Delete(ctx context.Context, id int) error
}

type EmployeeServer struct {
	atlinkspbconnect.UnimplementedEmployeeServiceHandler
	store employeeStore
	audit *audit.Service
}

func NewEmployeeServer(store employeeStore, audit *audit.Service) *EmployeeServer {
	return &EmployeeServer{store: store, audit: audit}
}

func (s *EmployeeServer) ListEmployees(ctx context.Context, req *connect.Request[pb.ListEmployeesRequest]) (*connect.Response[pb.ListEmployeesResponse], error) {
	filter := protoToEmployeeFilter(req.Msg)
	result, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list employees: %w", err))
	}

	employees := make([]*pb.Employee, len(result.Items))
	for i := range result.Items {
		employees[i] = employeeToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListEmployeesResponse{
		Employees: employees,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *EmployeeServer) GetEmployee(ctx context.Context, req *connect.Request[pb.GetEmployeeRequest]) (*connect.Response[pb.GetEmployeeResponse], error) {
	e, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("employee %d not found", req.Msg.Id))
	}
	return connect.NewResponse(&pb.GetEmployeeResponse{Employee: employeeToProto(e)}), nil
}

func (s *EmployeeServer) CreateEmployee(ctx context.Context, req *connect.Request[pb.CreateEmployeeRequest]) (*connect.Response[pb.CreateEmployeeResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	e := createEmployeeReqToModel(req.Msg)
	if err := s.store.Create(ctx, e); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create employee: %w", err))
	}

	s.audit.Log(ctx, "employees", e.ID, "INSERT", nil, e)

	return connect.NewResponse(&pb.CreateEmployeeResponse{Employee: employeeToProto(e)}), nil
}

func (s *EmployeeServer) UpdateEmployee(ctx context.Context, req *connect.Request[pb.UpdateEmployeeRequest]) (*connect.Response[pb.UpdateEmployeeResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("employee %d not found", req.Msg.Id))
	}

	e := updateEmployeeReqToModel(req.Msg)
	if err := s.store.Update(ctx, e); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update employee: %w", err))
	}

	s.audit.Log(ctx, "employees", e.ID, "UPDATE", old, e)

	updated, err := s.store.GetByID(ctx, e.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated employee: %w", err))
	}
	return connect.NewResponse(&pb.UpdateEmployeeResponse{Employee: employeeToProto(updated)}), nil
}

func (s *EmployeeServer) DeleteEmployee(ctx context.Context, req *connect.Request[pb.DeleteEmployeeRequest]) (*connect.Response[pb.DeleteEmployeeResponse], error) {
	if err := s.store.Delete(ctx, int(req.Msg.Id)); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("employee %d not found", req.Msg.Id))
	}
	s.audit.Log(ctx, "employees", int(req.Msg.Id), "DELETE", nil, nil)
	return connect.NewResponse(&pb.DeleteEmployeeResponse{Success: true}), nil
}

// --- Employee converters ---

func employeeToProto(e *models.Employee) *pb.Employee {
	return &pb.Employee{
		Id:                   int32(e.ID),
		Name:                 e.Name,
		Address:              sp(e.Address),
		Address2:             sp(e.Address2),
		City:                 sp(e.City),
		State:                sp(e.State),
		Zip:                  sp(e.Zip),
		Phone:                sp(e.Phone),
		Rate:                 sp(e.Rate),
		RateCalcType:         sp(e.RateCalcType),
		Active:               e.Active,
		IsDriver:             e.IsDriver,
		IsSales:              e.IsSales,
		EmpIdNumber:          sp(e.EmpIDNumber),
		EmploymentDate:       timeStr(e.EmploymentDate),
		TerminationDate:      timeStr(e.TerminationDate),
		EmergencyContact:     sp(e.EmergencyContact),
		EmergencyPhone:       sp(e.EmergencyPhone),
		DriversLicenseNumber: sp(e.DriversLicenseNumber),
		DriversLicenseState:  sp(e.DriversLicenseState),
		CopyOfCdl:            e.CopyOfCDL,
		CdlExp:               timeStr(e.CDLExp),
		CopyOfMedCert:        e.CopyOfMedCert,
		MedCertExp:           timeStr(e.MedCertExp),
		DotApplication:       e.DOTApplication,
		DotApplicationExp:    timeStr(e.DOTApplicationExp),
		Reserve:              sp(e.Reserve),
		AddRate:              sp(e.AddRate),
		AddRateCalcType:      sp(e.AddRateCalcType),
		Username:             sp(e.Username),
		CreatedAt:            e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            e.UpdatedAt.Format(time.RFC3339),
	}
}

func protoToEmployeeFilter(msg *pb.ListEmployeesRequest) models.EmployeeFilter {
	f := models.EmployeeFilter{}
	if msg.Pagination != nil {
		f.Page = int(msg.Pagination.Page)
		f.PageSize = int(msg.Pagination.PageSize)
	}
	if msg.Search != nil {
		f.Search = *msg.Search
	}
	if msg.Active != nil {
		f.Active = *msg.Active
	}
	if msg.IsDriver != nil {
		f.IsDriver = *msg.IsDriver
	}
	return f
}

func createEmployeeReqToModel(msg *pb.CreateEmployeeRequest) *models.Employee {
	return &models.Employee{
		Name:                 msg.Name,
		Address:              sp(msg.Address),
		Address2:             sp(msg.Address2),
		City:                 sp(msg.City),
		State:                sp(msg.State),
		Zip:                  sp(msg.Zip),
		Phone:                sp(msg.Phone),
		Rate:                 sp(msg.Rate),
		RateCalcType:         sp(msg.RateCalcType),
		Active:               msg.Active,
		IsDriver:             msg.IsDriver,
		IsSales:              msg.IsSales,
		EmpIDNumber:          sp(msg.EmpIdNumber),
		EmploymentDate:       parseDate(msg.EmploymentDate),
		EmergencyContact:     sp(msg.EmergencyContact),
		EmergencyPhone:       sp(msg.EmergencyPhone),
		DriversLicenseNumber: sp(msg.DriversLicenseNumber),
		DriversLicenseState:  sp(msg.DriversLicenseState),
	}
}

func updateEmployeeReqToModel(msg *pb.UpdateEmployeeRequest) *models.Employee {
	return &models.Employee{
		ID:                   int(msg.Id),
		Name:                 msg.Name,
		Address:              sp(msg.Address),
		Address2:             sp(msg.Address2),
		City:                 sp(msg.City),
		State:                sp(msg.State),
		Zip:                  sp(msg.Zip),
		Phone:                sp(msg.Phone),
		Rate:                 sp(msg.Rate),
		RateCalcType:         sp(msg.RateCalcType),
		Active:               msg.Active,
		IsDriver:             msg.IsDriver,
		IsSales:              msg.IsSales,
		EmpIDNumber:          sp(msg.EmpIdNumber),
		EmploymentDate:       parseDate(msg.EmploymentDate),
		TerminationDate:      parseDate(msg.TerminationDate),
		EmergencyContact:     sp(msg.EmergencyContact),
		EmergencyPhone:       sp(msg.EmergencyPhone),
		DriversLicenseNumber: sp(msg.DriversLicenseNumber),
		DriversLicenseState:  sp(msg.DriversLicenseState),
	}
}
