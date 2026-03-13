package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"

	pb "github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1"
)

func registerFeedbackTools(register toolRegister, client *atlClient) {
	register(mcp.NewTool("list_feedback",
		mcp.WithDescription("List feedback items with optional filters. Super admins see all feedback; other users see only their company's feedback."),
		mcp.WithString("status", mcp.Description("Filter by status: 'open', 'reviewed', 'closed', 'active' (open+reviewed), or 'all' (default: all)")),
		mcp.WithString("category", mcp.Description("Filter by category: 'feature', 'bug', 'question', 'other'")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("page_size", mcp.Description("Items per page (default: 25)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.feedback.ListFeedback(ctx, connect.NewRequest(&pb.ListFeedbackRequest{
			Pagination: &pb.PaginationRequest{
				Page:     int32(argInt(args, "page", 1)),
				PageSize: int32(argInt(args, "page_size", 25)),
			},
			Status:   argStrPtr(args, "status"),
			Category: argStrPtr(args, "category"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatFeedbackList(resp.Msg)), nil
	})

	register(mcp.NewTool("get_feedback",
		mcp.WithDescription("Get a single feedback item by ID, including its comments. Admins see internal comments; regular users do not."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Feedback ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.feedback.GetFeedback(ctx, connect.NewRequest(&pb.GetFeedbackRequest{
			Id: int32(argInt(args, "id", 0)),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(formatFeedbackDetail(resp.Msg)), nil
	})

	register(mcp.NewTool("create_feedback",
		mcp.WithDescription("Submit new feedback (bug report, feature request, question, etc.)."),
		mcp.WithString("message", mcp.Required(), mcp.Description("Feedback message")),
		mcp.WithString("category", mcp.Description("Category: 'feature', 'bug', 'question', 'other' (default: other)")),
		mcp.WithString("page_url", mcp.Description("URL of the page this feedback is about")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.feedback.CreateFeedback(ctx, connect.NewRequest(&pb.CreateFeedbackRequest{
			Message:  argStr(args, "message"),
			Category: argStrPtr(args, "category"),
			PageUrl:  argStrPtr(args, "page_url"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		fb := resp.Msg.Feedback
		return mcp.NewToolResultText(fmt.Sprintf("Created feedback #%d (%s) — status: %s", fb.Id, fb.Category, fb.Status)), nil
	})

	register(mcp.NewTool("update_feedback_status",
		mcp.WithDescription("Update the status of a feedback item."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Feedback ID")),
		mcp.WithString("status", mcp.Required(), mcp.Description("New status: 'open', 'reviewed', or 'closed'")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.feedback.UpdateFeedbackStatus(ctx, connect.NewRequest(&pb.UpdateFeedbackStatusRequest{
			Id:     int32(argInt(args, "id", 0)),
			Status: argStr(args, "status"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Feedback #%d status updated to: %s", resp.Msg.Feedback.Id, resp.Msg.Feedback.Status)), nil
	})

	register(mcp.NewTool("add_feedback_comment",
		mcp.WithDescription("Add a comment to a feedback item. Set internal=true for staff-only notes."),
		mcp.WithNumber("feedback_id", mcp.Required(), mcp.Description("Feedback ID to comment on")),
		mcp.WithString("message", mcp.Required(), mcp.Description("Comment text")),
		mcp.WithBoolean("internal", mcp.Description("If true, comment is only visible to admins (default: false)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argMap(req)
		resp, err := client.feedback.AddComment(ctx, connect.NewRequest(&pb.AddCommentRequest{
			FeedbackId: int32(argInt(args, "feedback_id", 0)),
			Message:    argStr(args, "message"),
			Internal:   argBool(args, "internal"),
		}))
		if err != nil {
			return mcp.NewToolResultError(connectErr(err)), nil
		}
		c := resp.Msg.Comment
		vis := "public"
		if c.Internal {
			vis = "internal"
		}
		return mcp.NewToolResultText(fmt.Sprintf("Added %s comment #%d to feedback #%d", vis, c.Id, c.FeedbackId)), nil
	})
}

func formatFeedbackList(resp *pb.ListFeedbackResponse) string {
	if len(resp.Items) == 0 {
		return "No feedback found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d feedback items (page %d, %d per page):\n\n",
		resp.Pagination.TotalCount, resp.Pagination.Page, resp.Pagination.PageSize))
	for _, fb := range resp.Items {
		sb.WriteString(fmt.Sprintf("  #%d [%s] %s — by %s (%s)",
			fb.Id, fb.Status, fb.Category, fb.Username, fb.CreatedAt[:10]))
		if fb.CompanyName != nil {
			sb.WriteString(fmt.Sprintf(" — %s", *fb.CompanyName))
		}
		if fb.CommentCount > 0 {
			sb.WriteString(fmt.Sprintf(" [%d comments]", fb.CommentCount))
		}
		sb.WriteString("\n")
		// Show truncated message
		msg := fb.Message
		if len(msg) > 80 {
			msg = msg[:80] + "..."
		}
		sb.WriteString(fmt.Sprintf("    %s\n", msg))
	}
	return sb.String()
}

func formatFeedbackDetail(resp *pb.GetFeedbackResponse) string {
	fb := resp.Feedback
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Feedback #%d\n", fb.Id))
	sb.WriteString(fmt.Sprintf("  Status: %s | Category: %s\n", fb.Status, fb.Category))
	sb.WriteString(fmt.Sprintf("  By: %s | Created: %s\n", fb.Username, fb.CreatedAt[:10]))
	if fb.CompanyName != nil {
		sb.WriteString(fmt.Sprintf("  Company: %s\n", *fb.CompanyName))
	}
	if fb.PageUrl != "" {
		sb.WriteString(fmt.Sprintf("  Page: %s\n", fb.PageUrl))
	}
	sb.WriteString(fmt.Sprintf("\n  %s\n", fb.Message))

	if len(resp.Comments) > 0 {
		sb.WriteString(fmt.Sprintf("\n  --- %d Comments ---\n", len(resp.Comments)))
		for _, c := range resp.Comments {
			vis := ""
			if c.Internal {
				vis = " [INTERNAL]"
			}
			sb.WriteString(fmt.Sprintf("  %s (%s)%s — %s\n    %s\n",
				c.Username, c.UserRole, vis, c.CreatedAt[:10], c.Message))
		}
	}
	return sb.String()
}
