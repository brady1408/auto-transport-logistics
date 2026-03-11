package connectrpc

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/auth"
	pb "github.com/brady1408/atlinks/internal/gen/atlinks/v1"
	"github.com/brady1408/atlinks/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/atlinks/internal/models"
)

type feedbackStore interface {
	List(ctx context.Context, f models.FeedbackFilter) (*models.FeedbackListResult, error)
	GetByID(ctx context.Context, id int) (*models.Feedback, error)
	Create(ctx context.Context, fb *models.Feedback) error
	Update(ctx context.Context, fb *models.Feedback) error
	ListComments(ctx context.Context, feedbackID int, includeInternal bool) ([]models.FeedbackComment, error)
	CreateComment(ctx context.Context, c *models.FeedbackComment) error
}

type FeedbackServer struct {
	atlinkspbconnect.UnimplementedFeedbackServiceHandler
	store feedbackStore
	audit *audit.Service
}

func NewFeedbackServer(store feedbackStore, audit *audit.Service) *FeedbackServer {
	return &FeedbackServer{store: store, audit: audit}
}

func (s *FeedbackServer) ListFeedback(ctx context.Context, req *connect.Request[pb.ListFeedbackRequest]) (*connect.Response[pb.ListFeedbackResponse], error) {
	f := models.FeedbackFilter{}
	if req.Msg.Pagination != nil {
		f.Page = int(req.Msg.Pagination.Page)
		f.PageSize = int(req.Msg.Pagination.PageSize)
	}
	if req.Msg.Status != nil {
		f.Status = *req.Msg.Status
	}
	if req.Msg.Category != nil {
		f.Category = *req.Msg.Category
	}

	result, err := s.store.List(ctx, f)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list feedback: %w", err))
	}

	items := make([]*pb.Feedback, len(result.Items))
	for i := range result.Items {
		items[i] = feedbackToProto(&result.Items[i])
	}

	return connect.NewResponse(&pb.ListFeedbackResponse{
		Items: items,
		Pagination: &pb.PaginationResponse{
			TotalCount: int32(result.TotalCount),
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
		},
	}), nil
}

func (s *FeedbackServer) GetFeedback(ctx context.Context, req *connect.Request[pb.GetFeedbackRequest]) (*connect.Response[pb.GetFeedbackResponse], error) {
	fb, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("feedback %d not found", req.Msg.Id))
	}

	// Determine whether to include internal comments based on role
	user, _ := auth.GetUser(ctx)
	includeInternal := user.Role == "super_admin" || user.Role == "company_admin"

	comments, err := s.store.ListComments(ctx, fb.ID, includeInternal)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list comments: %w", err))
	}

	pbComments := make([]*pb.FeedbackComment, len(comments))
	for i := range comments {
		pbComments[i] = commentToProto(&comments[i])
	}

	return connect.NewResponse(&pb.GetFeedbackResponse{
		Feedback: feedbackToProto(fb),
		Comments: pbComments,
	}), nil
}

func (s *FeedbackServer) CreateFeedback(ctx context.Context, req *connect.Request[pb.CreateFeedbackRequest]) (*connect.Response[pb.CreateFeedbackResponse], error) {
	if req.Msg.Message == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message is required"))
	}

	user, _ := auth.GetUser(ctx)

	category := "other"
	if req.Msg.Category != nil && *req.Msg.Category != "" {
		category = *req.Msg.Category
	}
	pageURL := ""
	if req.Msg.PageUrl != nil {
		pageURL = *req.Msg.PageUrl
	}

	fb := &models.Feedback{
		UserID:   user.ID,
		Username: user.Username,
		PageURL:  pageURL,
		Category: category,
		Message:  req.Msg.Message,
	}

	if err := s.store.Create(ctx, fb); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create feedback: %w", err))
	}

	s.audit.Log(ctx, "feedback", fb.ID, "INSERT", nil, fb)

	return connect.NewResponse(&pb.CreateFeedbackResponse{
		Feedback: feedbackToProto(fb),
	}), nil
}

func (s *FeedbackServer) UpdateFeedbackStatus(ctx context.Context, req *connect.Request[pb.UpdateFeedbackStatusRequest]) (*connect.Response[pb.UpdateFeedbackStatusResponse], error) {
	if req.Msg.Status == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("status is required"))
	}

	old, err := s.store.GetByID(ctx, int(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("feedback %d not found", req.Msg.Id))
	}

	fb := &models.Feedback{
		ID:     int(req.Msg.Id),
		Status: req.Msg.Status,
	}
	if err := s.store.Update(ctx, fb); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update feedback: %w", err))
	}

	s.audit.Log(ctx, "feedback", fb.ID, "UPDATE", old, fb)

	updated, err := s.store.GetByID(ctx, fb.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch updated feedback: %w", err))
	}
	return connect.NewResponse(&pb.UpdateFeedbackStatusResponse{
		Feedback: feedbackToProto(updated),
	}), nil
}

func (s *FeedbackServer) ListComments(ctx context.Context, req *connect.Request[pb.ListCommentsRequest]) (*connect.Response[pb.ListCommentsResponse], error) {
	comments, err := s.store.ListComments(ctx, int(req.Msg.FeedbackId), req.Msg.IncludeInternal)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list comments: %w", err))
	}

	pbComments := make([]*pb.FeedbackComment, len(comments))
	for i := range comments {
		pbComments[i] = commentToProto(&comments[i])
	}

	return connect.NewResponse(&pb.ListCommentsResponse{Comments: pbComments}), nil
}

func (s *FeedbackServer) AddComment(ctx context.Context, req *connect.Request[pb.AddCommentRequest]) (*connect.Response[pb.AddCommentResponse], error) {
	if req.Msg.Message == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message is required"))
	}

	user, _ := auth.GetUser(ctx)

	c := &models.FeedbackComment{
		FeedbackID: int(req.Msg.FeedbackId),
		UserID:     user.ID,
		Username:   user.Username,
		UserRole:   user.Role,
		CompanyID:  user.CompanyID,
		Message:    req.Msg.Message,
		Internal:   req.Msg.Internal,
	}

	if err := s.store.CreateComment(ctx, c); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("add comment: %w", err))
	}

	s.audit.Log(ctx, "feedback_comments", c.ID, "INSERT", nil, c)

	return connect.NewResponse(&pb.AddCommentResponse{
		Comment: commentToProto(c),
	}), nil
}

func feedbackToProto(fb *models.Feedback) *pb.Feedback {
	p := &pb.Feedback{
		Id:           int32(fb.ID),
		UserId:       int32(fb.UserID),
		Username:     fb.Username,
		PageUrl:      fb.PageURL,
		Category:     fb.Category,
		Message:      fb.Message,
		Status:       fb.Status,
		CommentCount: int32(fb.CommentCount),
		CreatedAt:    fb.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    fb.UpdatedAt.Format(time.RFC3339),
	}
	if fb.CompanyName != "" {
		p.CompanyName = &fb.CompanyName
	}
	return p
}

func commentToProto(c *models.FeedbackComment) *pb.FeedbackComment {
	return &pb.FeedbackComment{
		Id:         int32(c.ID),
		FeedbackId: int32(c.FeedbackID),
		UserId:     int32(c.UserID),
		Username:   c.Username,
		UserRole:   c.UserRole,
		Message:    c.Message,
		Internal:   c.Internal,
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
	}
}
