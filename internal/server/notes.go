package server

// noteWritingPrompt returns the note-writing best practices guide.
// Content is built from string concatenation to avoid raw-string backtick
// conflicts with Xurrent's inline code formatting syntax.
func noteWritingPrompt() string {
	return "" +
		"You are writing a note in Xurrent. Follow these guidelines for professional, useful notes.\n" +
		"\n" +
		"## Default: Internal Notes\n" +
		"\n" +
		"**Always set internal:true unless explicitly told otherwise.** Internal notes are visible only to specialists. Public notes go to requesters.\n" +
		"\n" +
		"Exceptions where public notes are appropriate:\n" +
		"- The user explicitly says \"tell the requester\" or \"make it public\"\n" +
		"- You are providing an update the requester needs to see\n" +
		"- The note is a communication TO the requester\n" +
		"\n" +
		"## Structure Guidelines\n" +
		"\n" +
		"### Be Concise\n" +
		"- One opening summary line (1-2 sentences)\n" +
		"- Details in bulleted lists where appropriate\n" +
		"- No fluff, no greetings (Hi, Hello), no signatures\n" +
		"- Maximum ~250 words unless providing detailed analysis\n" +
		"\n" +
		"### Use Headers for Sections\n" +
		"Use ## for major sections (e.g. ## Analysis, ## Root Cause, ## Resolution)\n" +
		"\n" +
		"### Format References Properly\n" +
		"**Xurrent records** — use record references so they appear as clickable links:\n" +
		"- Requests: <request#12345>\n" +
		"- Problems: <problem#678>\n" +
		"- Workflows: <workflow#901>\n" +
		"- Tasks: <task#234>\n" +
		"- Configuration items: <ci#567>\n" +
		"- Services: <service#89>\n" +
		"- SLAs: <sla#12>\n" +
		"- Teams: <team#34>\n" +
		"- People: <person#56>\n" +
		"- Organizations: <organization#78>\n" +
		"- Knowledge articles: <knowledge_article#90>\n" +
		"\n" +
		"**Person mentions** — use @mentions to notify people:\n" +
		"- <@id|Display Name> (get the id from a person query, name from the record)\n" +
		"\n" +
		"**External links** — use markdown links:\n" +
		"- [Xurrent docs](https://developer.xurrent.com/v1/)\n" +
		"\n" +
		"### Format Data and Logs\n" +
		"**Short values** — use backtick monospace (the backtick character around values):\n" +
		"  The error was `ERR_TIMEOUT_402` on server `prod-web-12`.\n" +
		"\n" +
		"**Log excerpts** — use triple-backtick code blocks (add blank lines above and below):\n" +
		"  ```\n" +
		"  2026-10-06 03:45:22 ERROR Connection refused\n" +
		"  2026-10-06 03:45:23 WARN Retrying in 5s\n" +
		"  ```\n" +
		"\n" +
		"**JSON output** — same triple-backtick code blocks:\n" +
		"  ```\n" +
		"  {\"status\": \"error\", \"code\": 503, \"message\": \"Service Unavailable\"}\n" +
		"  ```\n" +
		"\n" +
		"### Use Formatting\n" +
		"- **Bold** for key findings and actions: **Root cause identified**\n" +
		"- __Italic__ for emphasis: __not__ a configuration issue\n" +
		"- +Underline+ for critical warnings: +This is a P1 incident+\n" +
		"- Numbered lists for procedures (1. step one)\n" +
		"- Bulleted lists for findings (* finding)\n" +
		"- > Quotes for content from external sources (email bodies, user messages)\n" +
		"\n" +
		"### Templates\n" +
		"\n" +
		"**Investigation Update (internal)**:\n" +
		"  ## Investigation Update\n" +
		"  Analyzed the **<subject>** affecting <entity>.\n" +
		"\n" +
		"  Findings:\n" +
		"  * <finding 1>\n" +
		"  * <finding 2>\n" +
		"\n" +
		"  Root cause: **<summary>**\n" +
		"\n" +
		"  Next steps:\n" +
		"  1. <action 1>\n" +
		"  2. <action 2>\n" +
		"\n" +
		"  Related: <request#ticket_id>\n" +
		"\n" +
		"**Resolution (internal)**:\n" +
		"  ## Resolution\n" +
		"  **<problem>** resolved via <method>.\n" +
		"\n" +
		"  The <component> was misconfigured. Updated <setting> from <old> to <new>. Verified by <verification>.\n" +
		"\n" +
		"  No further action required.\n" +
		"\n" +
		"**Communication to Requester (public)**:\n" +
		"  Hi <name>,\n" +
		"\n" +
		"  We've resolved the issue with <subject>. <explanation>.\n" +
		"\n" +
		"  <next steps or what to expect>\n" +
		"\n" +
		"  If you have questions, reply to this notification.\n" +
		"\n" +
		"### Attachments\n" +
		"Consider attaching relevant files when they add value:\n" +
		"- Log files for error investigations → add as attachment\n" +
		"- Screenshots of configuration → add as attachment\n" +
		"- Long terminal output → attach rather than inline\n" +
		"\n" +
		"Use the file-attachment skill for upload instructions.\n" +
		"\n" +
		"### Don'ts\n" +
		"- Don't write \"Note:\" or \"Internal Note:\" prefixes (the UI shows this)\n" +
		"- Don't include greetings or signatures\n" +
		"- Don't repeat information already visible in the request fields\n" +
		"- Don't write in ALL CAPS\n" +
		"- Don't use bare URLs (always wrap in [text](url))\n" +
		"- Don't use HTML (use Xurrent formatting syntax)\n"
}
