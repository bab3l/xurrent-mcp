package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources() {
	s.Server.AddResource(&mcp.Resource{
		URI:         "xurrent://domain/graph",
		Name:        "Xurrent Domain Graph",
		Description: "Compact entity relationship map showing how Xurrent objects relate.",
		MIMEType:    "application/json",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: "xurrent://domain/graph", MIMEType: "application/json", Text: domainGraphJSON()},
			},
		}, nil
	})

	s.Server.AddResource(&mcp.Resource{
		URI:         "xurrent://docs/index",
		Name:        "Xurrent Documentation Index",
		Description: "Index of Xurrent developer documentation URLs by topic.",
		MIMEType:    "application/json",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: "xurrent://docs/index", MIMEType: "application/json", Text: docsIndexJSON()},
			},
		}, nil
	})

	s.Server.AddResource(&mcp.Resource{
		URI:         "xurrent://docs/ui-extensions",
		Name:        "UI Extensions Guide",
		Description: "Xurrent UI Extension JS API, CSS constraints, and development patterns.",
		MIMEType:    "text/markdown",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: "xurrent://docs/ui-extensions", MIMEType: "text/markdown", Text: uiExtensionsGuide()},
			},
		}, nil
	})

	s.Server.AddResource(&mcp.Resource{
		URI:         "xurrent://docs/automation",
		Name:        "Automation Rules Guide",
		Description: "Xurrent Automation Rules expression syntax, triggers, actions, and best practices.",
		MIMEType:    "text/markdown",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: "xurrent://docs/automation", MIMEType: "text/markdown", Text: automationGuide()},
			},
		}, nil
	})
}

func (s *Server) registerPrompts() {
	s.Server.AddPrompt(&mcp.Prompt{
		Name:        "itsm-relationships",
		Description: "Explains how Xurrent ITSM entities relate: services, SLAs, CIs, requests, workflows, problems, and their dependencies.",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "ITSM Entity Relationships in Xurrent",
			Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.TextContent{Text: itsmRelationshipsPrompt()}},
			},
		}, nil
	})

	s.Server.AddPrompt(&mcp.Prompt{
		Name:        "ui-extension-development",
		Description: "Guidance for developing Xurrent UI Extensions: HTML/CSS/JS constraints, version management, and common patterns.",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "UI Extension Development Guide",
			Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.TextContent{Text: uiExtensionDevPrompt()}},
			},
		}, nil
	})

	s.Server.AddPrompt(&mcp.Prompt{
		Name:        "automation-rule-development",
		Description: "Guidance for creating Xurrent Automation Rules: expression syntax, triggers, conditions, actions, and testing.",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Automation Rule Development Guide",
			Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.TextContent{Text: automationDevPrompt()}},
			},
		}, nil
	})

	s.Server.AddPrompt(&mcp.Prompt{
		Name:        "email-triage",
		Description: "Step-by-step guide for investigating inbound email issues in Xurrent: why tickets are being created when they should be blocked, automation rule analysis, and email source tracing.",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Email Triage Investigation Guide",
			Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.TextContent{Text: emailTriagePrompt()}},
			},
		}, nil
	})

	s.Server.AddPrompt(&mcp.Prompt{
		Name:        "audit-trail",
		Description: "How to use Xurrent audit entries to investigate who changed what and when. Covers top-level audit_lines, per-entity audit (request, automation rule), and using audit data for incident investigation.",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Audit Trail Investigation Guide",
			Messages: []*mcp.PromptMessage{
				{Role: "user", Content: &mcp.TextContent{Text: auditTrailPrompt()}},
			},
		}, nil
	})
}
