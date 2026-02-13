# Slack Integration Guide

## Overview

This document explains how to integrate Slack with the SDS Integration Dashboard using the Apps Platform Slack integration.

## Architecture

```
Slack API → Slack Proxy (edge) → Your App (Cloud Run)
```

The Slack Proxy handles:
- Request signature verification
- Token management and refresh
- Event routing to your app

## Quick Start

### 1. Enable Slack

Add to `project.toml`:
```toml
enable_slack = true
```

### 2. Request Installation

Visit:
```
https://apps.applied.dev/slack/install?app=sds-integration-dashboard
```

This automatically notifies SamK for approval. Once approved, the bot token is stored in Secret Manager.

### 3. Add Code

```go
import "github.com/applied/slacklib"

// Auto-fetches token from Secret Manager
bot, _ := slacklib.New(slacklib.Config{})

// Register routes at /slack
bot.RegisterRoutes(r.Group("/slack"))

// Handle mentions
bot.OnMention(func(e slacklib.MentionEvent) {
    bot.Reply(e, "Hello!")
})

// Send notifications
bot.PostMessage(slacklib.PostMessageRequest{
    Channel: "C12345678",
    Text: "⚠️ KPI Alert: Data Collection Efficiency dropped to 72%",
})
```

### 4. Request Channel Access

Contact **SamK** to add your bot to specific channels. Only non-confidential channels in the "Applied Bots" workspace.

## What Apps Platform Handles Automatically

✅ Token management (stored in Secret Manager)
✅ Signature verification (via Slack Proxy)
✅ Token rotation
✅ Automatic routes: `/slack/events`, `/slack/commands`, `/slack/interactions`

## Token Types

- **Bot Tokens** (`xoxb-...`) - App-specific operations (most common)
- **User Tokens** (`xoxp-...`) - User-specific actions
- **App-level Tokens** (`xapp-...`) - App-level operations

Apps Platform uses Bot tokens by default.

## Use Cases for This Dashboard

### 1. KPI Alerts

Send automated alerts when metrics fall below targets:

```go
if efficiency < 95.0 {
    bot.PostMessage(slacklib.PostMessageRequest{
        Channel: channelID,
        Text: fmt.Sprintf("⚠️ Data Collection Efficiency: %.1f%%", efficiency),
    })
}
```

### 2. Daily/Weekly Reports

Post summary reports using Block Kit:

```go
bot.PostMessage(slacklib.PostMessageRequest{
    Channel: channelID,
    Blocks: []slacklib.Block{
        {Type: "header", Text: "📊 Weekly KPI Summary"},
        {Type: "section", Text: "*#1 Time in Build*: 4.5 days ✅"},
        {Type: "section", Text: "*#3 MTBF*: 12 failures ⚠️"},
        {Type: "section", Text: "*#7 Data Collection*: 72% ⚠️"},
    },
})
```

### 3. Slash Commands

Create commands for on-demand queries:

```go
bot.Command("/kpi", func(e slacklib.CommandEvent) {
    // Fetch current KPI data
    response := "📊 Current KPIs:\n"
    response += "• Time in Build: 4.5 days\n"
    response += "• MTBF: 12 failures\n"
    bot.Respond(e, response)
})
```

### 4. Interactive Buttons

Add buttons for common actions:

```go
bot.PostMessage(slacklib.PostMessageRequest{
    Channel: channelID,
    Blocks: []slacklib.Block{
        {Type: "section", Text: "Deployment failed. Retry?"},
        {Type: "actions", Elements: []slacklib.Element{
            {Type: "button", Text: "Retry", ActionID: "retry_deploy"},
        }},
    },
})

bot.Action("retry_deploy", func(e slacklib.ActionEvent) {
    // Trigger deployment
})
```

## Message Formatting

### Text Formatting (mrkdwn)

```
*bold* _italic_ `code` ~strikethrough~
<https://example.com|link text>
<@U12345678> mention user
<#C12345678> mention channel
```

### Block Kit (Rich Layouts)

Use JSON-based UI components:
- Headers, sections, dividers
- Buttons, select menus, date pickers
- Images, context blocks

Visual builder: https://app.slack.com/block-kit-builder

## API Methods Reference

Common methods from the official Slack API:

### Chat
- `chat.postMessage` - Send a message
- `chat.update` - Update a message
- `chat.delete` - Delete a message

### Conversations
- `conversations.list` - List channels
- `conversations.info` - Get channel info
- `conversations.members` - List channel members

### Users
- `users.list` - List users
- `users.info` - Get user info

### Reactions
- `reactions.add` - Add emoji reaction
- `reactions.remove` - Remove reaction

## Security & Constraints

### Separate Workspace
- Bots operate in "Applied Bots" workspace
- Only non-confidential channels
- No customer data channels ever exposed

### Rate Limits
- 1 message per second per channel
- Varies by API method
- Batch operations if hitting limits

### DRI Responsibilities
1. Ensure bot operates only in appropriate channels
2. Validate no customer data flows through bot channels
3. Request channel access only for legitimate use cases

## Routes Created by slacklib

When you call `bot.RegisterRoutes(r.Group("/slack"))`:

| Route | Purpose | Handler |
|-------|---------|---------|
| POST /slack/events | Mentions, messages | OnMention, OnDM |
| POST /slack/commands | Slash commands | Command |
| POST /slack/interactions | Buttons, modals | Action, ViewSubmission |

## Customization

### Bot Profile
Customize at api.slack.com/apps:
- App name
- Icon (512x512 PNG recommended)
- Description
- Background color

### Add Slash Commands
1. Go to api.slack.com/apps → your app
2. Navigate to Slash Commands → Create New Command
3. Set Request URL: `https://sds-integration-dashboard.experimental.apps.applied.dev/slack/commands`

### Add Shortcuts
1. Go to Interactivity & Shortcuts
2. Enable Interactivity
3. Add shortcut with callback ID

### Request Additional Scopes
1. Go to api.slack.com/apps → your app
2. OAuth & Permissions → Bot Token Scopes
3. Add required scopes
4. Reinstall app

## Troubleshooting

### Events Not Arriving
- Check `enable_slack = true` in project.toml
- Verify app is deployed and running
- Ensure routes registered correctly: `bot.RegisterRoutes(r.Group("/slack"))`
- Check Event Subscriptions enabled at api.slack.com/apps

### Token Errors
- Verify installation completed
- Check Secret Manager for `sds-integration-dashboard-slack-bot-token`
- Redeploy to pick up new token

### Rate Limit Errors
- Add delays between messages (1 msg/sec per channel)
- Batch operations when possible
- Use threading to group related messages

## Resources

- **Apps Platform Docs**: https://apps.applied.dev/docs/slack-proxy
- **slacklib Reference**: Internal Applied library documentation
- **Official Slack API**: https://docs.slack.dev/
- **Block Kit Builder**: https://app.slack.com/block-kit-builder
- **Slack Scopes Reference**: https://api.slack.com/scopes

## Next Steps

To enable Slack for this dashboard:

1. Add `enable_slack = true` to project.toml
2. Request installation and get SamK's approval
3. Add slacklib code to send KPI alerts
4. Deploy and test
5. Request channel access from SamK

Example first implementation: Send alert when Data Collection Efficiency drops below 95%.
