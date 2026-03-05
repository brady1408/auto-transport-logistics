import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

const API_URL = process.env.ATLINKS_API_URL || "https://atlinks.app";
const API_KEY = process.env.ATLINKS_API_KEY;

if (!API_KEY) {
  console.error("ATLINKS_API_KEY is required");
  process.exit(1);
}

async function api(path, options = {}) {
  const url = `${API_URL}${path}`;
  const res = await fetch(url, {
    ...options,
    headers: {
      "X-API-Key": API_KEY,
      "Content-Type": "application/json",
      ...options.headers,
    },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API ${res.status}: ${text}`);
  }
  return res.json();
}

const server = new McpServer({
  name: "atlinks-feedback",
  version: "1.0.0",
});

server.tool(
  "create_feedback",
  "Create a new feedback item (feature request, bug report, or note)",
  {
    category: z
      .enum(["bug", "feature", "question", "other"])
      .default("feature")
      .describe("Category of the feedback"),
    message: z.string().min(1).describe("The feedback message or feature description"),
    page_url: z.string().optional().describe("Optional URL or context reference"),
  },
  async ({ category, message, page_url }) => {
    const body = { category, message };
    if (page_url) body.page_url = page_url;

    const data = await api("/api/feedback", {
      method: "POST",
      body: JSON.stringify(body),
    });
    return {
      content: [{ type: "text", text: `Feedback #${data.id} created (${category}).` }],
    };
  }
);

server.tool(
  "list_feedback",
  "List feedback items with optional filters",
  {
    status: z
      .enum(["open", "reviewed", "closed", "all"])
      .default("open")
      .describe("Filter by status"),
    category: z
      .string()
      .optional()
      .describe("Filter by category (bug, feature, question, other)"),
    page: z.number().int().min(1).default(1).describe("Page number"),
    page_size: z.number().int().min(1).max(100).default(25).describe("Items per page"),
  },
  async ({ status, category, page, page_size }) => {
    const params = new URLSearchParams({ status, page: String(page), page_size: String(page_size) });
    if (category) params.set("category", category);

    const data = await api(`/api/feedback?${params}`);
    const items = data.items || [];

    if (items.length === 0) {
      return { content: [{ type: "text", text: `No feedback items found (status=${status}).` }] };
    }

    const lines = items.map(
      (f) =>
        `#${f.id} [${f.status}] ${f.category} — ${f.username} (${f.company_name || "unknown"})` +
        `\n  "${f.message.slice(0, 120)}${f.message.length > 120 ? "..." : ""}"` +
        `\n  Page: ${f.page_url || "n/a"} | Comments: ${f.comment_count} | ${f.created_at}`
    );

    const text = `Feedback (${data.total} total, page ${data.page}):\n\n${lines.join("\n\n")}`;
    return { content: [{ type: "text", text }] };
  }
);

server.tool(
  "get_feedback",
  "Get a single feedback item with all comments",
  {
    id: z.number().int().describe("Feedback item ID"),
  },
  async ({ id }) => {
    const data = await api(`/api/feedback/${id}`);
    const fb = data.feedback;
    const comments = data.comments || [];

    let text =
      `Feedback #${fb.id}\n` +
      `Status: ${fb.status} | Category: ${fb.category}\n` +
      `From: ${fb.username} (company ${fb.company_name || fb.company_id})\n` +
      `Page: ${fb.page_url || "n/a"}\n` +
      `Created: ${fb.created_at}\n\n` +
      `Message:\n${fb.message}`;

    if (comments.length > 0) {
      text += "\n\n--- Comments ---\n";
      for (const c of comments) {
        const tag = c.internal ? " [INTERNAL]" : "";
        text += `\n${c.username} (${c.user_role})${tag} — ${c.created_at}\n${c.message}\n`;
      }
    } else {
      text += "\n\nNo comments yet.";
    }

    return { content: [{ type: "text", text }] };
  }
);

server.tool(
  "reply_to_feedback",
  "Post a comment on a feedback item",
  {
    id: z.number().int().describe("Feedback item ID"),
    message: z.string().min(1).describe("Comment text"),
    internal: z
      .boolean()
      .default(false)
      .describe("If true, comment is internal (not visible to submitter)"),
  },
  async ({ id, message, internal }) => {
    await api(`/api/feedback/${id}/comments`, {
      method: "POST",
      body: JSON.stringify({ message, internal }),
    });
    return {
      content: [
        {
          type: "text",
          text: `Comment posted on feedback #${id}${internal ? " (internal)" : ""}.`,
        },
      ],
    };
  }
);

server.tool(
  "update_feedback_status",
  "Update the status of a feedback item",
  {
    id: z.number().int().describe("Feedback item ID"),
    status: z.enum(["open", "reviewed", "closed"]).describe("New status"),
  },
  async ({ id, status }) => {
    await api(`/api/feedback/${id}`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    });
    return {
      content: [{ type: "text", text: `Feedback #${id} status updated to "${status}".` }],
    };
  }
);

server.tool(
  "get_activity_stats",
  "Get aggregated activity stats (request volume, active users, top pages, recent logins). Use hours=24 for the last day, hours=168 for the last week.",
  {
    hours: z
      .number()
      .int()
      .min(1)
      .max(720)
      .default(24)
      .describe("Number of hours to look back (1–720)"),
  },
  async ({ hours }) => {
    const data = await api(`/api/activity/stats?hours=${hours}`);

    const lines = [
      `Activity stats — last ${hours}h (since ${new Date(data.since).toLocaleString()})`,
      `  Total requests : ${data.total_requests}`,
      `  Unique users   : ${data.unique_users}`,
    ];

    if (data.active_users?.length) {
      lines.push("\nActive users:");
      for (const u of data.active_users) {
        lines.push(`  ${u.username} — ${u.count} requests`);
      }
    }

    if (data.top_paths?.length) {
      lines.push("\nTop pages:");
      for (const p of data.top_paths) {
        lines.push(`  ${p.path} — ${p.count}`);
      }
    }

    if (data.recent_logins?.length) {
      lines.push("\nRecent logins (first request per user per day):");
      for (const l of data.recent_logins) {
        lines.push(
          `  ${l.username ?? "unknown"} — ${new Date(l.created_at).toLocaleString()} from ${l.ip_address ?? "?"}`
        );
      }
    }

    return { content: [{ type: "text", text: lines.join("\n") }] };
  }
);

server.tool(
  "list_activity",
  "List raw activity log entries with optional filters. Useful for investigating what a specific user did or checking traffic on a path.",
  {
    user_id: z.number().int().optional().describe("Filter by user ID"),
    path: z.string().optional().describe("Filter by path substring"),
    date_from: z.string().optional().describe("ISO date/datetime lower bound"),
    date_to: z.string().optional().describe("ISO date/datetime upper bound"),
    page: z.number().int().min(1).default(1).describe("Page number"),
    page_size: z.number().int().min(1).max(200).default(50).describe("Items per page"),
  },
  async ({ user_id, path, date_from, date_to, page, page_size }) => {
    const params = new URLSearchParams({ page: String(page), page_size: String(page_size) });
    if (user_id) params.set("user_id", String(user_id));
    if (path) params.set("path", String(path));
    if (date_from) params.set("date_from", date_from);
    if (date_to) params.set("date_to", date_to);

    const data = await api(`/api/activity?${params}`);
    const items = data.items || [];

    if (items.length === 0) {
      return { content: [{ type: "text", text: "No activity found matching those filters." }] };
    }

    const lines = items.map(
      (l) =>
        `${new Date(l.created_at).toLocaleString()}  ${l.method} ${l.path}  ${l.status_code}  ${l.duration_ms}ms` +
        `  user=${l.username ?? "anon"}  ip=${l.ip_address ?? "?"}`
    );

    const text = `Activity (${data.total} total, page ${data.page}):\n\n${lines.join("\n")}`;
    return { content: [{ type: "text", text }] };
  }
);

const transport = new StdioServerTransport();
await server.connect(transport);
