package server

import (
	"fmt"
	"strings"
)

// validateAutomationSyntax checks automation rule expressions, condition, and actions for common errors.
func validateAutomationSyntax(expressions, condition, actions, trigger, generic string) []string {
	var issues []string

	// Parse named expressions to check references.
	exprNames := map[string]bool{}
	if strings.TrimSpace(expressions) != "" {
		for _, line := range strings.Split(expressions, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if idx := strings.Index(line, ":"); idx > 0 {
				name := strings.TrimSpace(line[:idx])
				if name != "" {
					exprNames[name] = true
				}
			}
		}
	}

	// Check actions are present if condition is defined.
	if strings.TrimSpace(actions) == "" && strings.TrimSpace(condition) != "" {
		issues = append(issues, "actions are empty but condition is defined — rule will evaluate but do nothing")
	}

	// Validate trigger format.
	if strings.TrimSpace(trigger) != "" {
		validTriggers := map[string]bool{
			"on status update": true, "on note added": true,
			"on create": true, "on update": true,
		}
		if !validTriggers[strings.ToLower(strings.TrimSpace(trigger))] {
			issues = append(issues, fmt.Sprintf("trigger %q may be invalid; common values: 'on status update', 'on note added', 'on create', 'on update'", trigger))
		}
	}

	// Validate generic values.
	if strings.TrimSpace(generic) != "" {
		validGeneric := map[string]bool{
			"request": true, "problem": true, "workflow": true,
			"task": true, "risk": true, "ci": true,
			"scim_user": true, "scim_group": true,
		}
		if !validGeneric[strings.ToLower(strings.TrimSpace(generic))] {
			issues = append(issues, fmt.Sprintf("generic %q is not a recognized record type", generic))
		}
	}

	// Warn about unused expressions.
	for name := range exprNames {
		if !strings.Contains(condition, name) && !strings.Contains(actions, name) {
			issues = append(issues, fmt.Sprintf("expression %q is defined but not used in condition or actions", name))
		}
	}

	return issues
}

// validateUIExtension checks HTML/CSS/JS for common issues in Xurrent UI Extensions.
func validateUIExtension(css, html, javaScript, category string) []string {
	var issues []string

	const maxSize = 64 * 1024 // 64KB per Xurrent docs.
	if len(css) > maxSize {
		issues = append(issues, fmt.Sprintf("CSS is %d bytes; Xurrent limit is 64KB (65536 bytes)", len(css)))
	}
	if len(html) > maxSize {
		issues = append(issues, fmt.Sprintf("HTML is %d bytes; Xurrent limit is 64KB (65536 bytes)", len(html)))
	}
	if len(javaScript) > maxSize {
		issues = append(issues, fmt.Sprintf("JavaScript is %d bytes; Xurrent limit is 64KB (65536 bytes)", len(javaScript)))
	}

	// Check for script tags in HTML (Xurrent uses js separately).
	if strings.Contains(strings.ToLower(html), "<script") {
		issues = append(issues, "HTML contains <script> tags — JavaScript should be in the javascript field, not inline in HTML")
	}

	// Check CSS uses scoped selectors.
	if strings.TrimSpace(css) != "" {
		hasScoped := false
		for _, line := range strings.Split(css, "\n") {
			if strings.Contains(line, ".") && strings.Contains(line, "{") {
				hasScoped = true
				break
			}
		}
		if !hasScoped {
			issues = append(issues, "CSS may lack scoped class selectors (e.g., .row.subject) — use row-level classes to avoid breaking page layout")
		}
	}

	// Category check.
	if strings.TrimSpace(category) != "" {
		validCats := map[string]bool{
			"request_template": true, "knowledge_article_template": true,
			"problem": true, "release": true, "workflow_template": true,
			"task_template": true, "project": true, "project_task_template": true,
			"service": true, "service_instance": true, "product": true,
			"product_category": true, "contract": true, "organization": true,
			"team": true, "person": true, "site": true, "risk": true,
			"custom_collection": true, "scim_user": true, "app_offering": true,
			"shop_article": true,
		}
		if !validCats[strings.TrimSpace(category)] {
			issues = append(issues, fmt.Sprintf("category %q is not a recognized Xurrent UI extension category", category))
		}
	}

	return issues
}
