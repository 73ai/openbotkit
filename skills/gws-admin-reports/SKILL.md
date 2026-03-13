---
name: gws-admin-reports
version: 1.0.0
description: "Google Workspace Admin SDK: Audit logs and usage reports."
metadata:
  openclaw:
    category: "productivity"
    requires:
      bins: ["gws"]
    cliHelp: "gws admin-reports --help"
---

# admin-reports (reports_v1)

> **PREREQUISITE:** Read `../gws-shared/SKILL.md` for auth, global flags, and security rules. If missing, run `gws generate-skills` to create it.

```bash
gws admin-reports <resource> <method> [flags]
```

## API Resources

### activities

  - `list` — Retrieves a list of activities for a specific customer's account and application such as the Admin console application or the Google Drive application. For more information, see the guides for administrator and Google Drive activity reports. For more information about the activity report's parameters, see the activity parameters reference guides.
    Required: `applicationName` (string), `userKey` (string)
    Key params: `actorIpAddress` (string), `applicationInfoFilter` (string), `customerId` (string), `endTime` (string), `eventName` (string)
    Example: `gws admin-reports activities list --params '{"applicationName":"...","userKey":"..."}'`
  - `watch` — Start receiving notifications for account activities. For more information, see Receiving Push Notifications.
    Required: `applicationName` (string), `userKey` (string)
    Key params: `actorIpAddress` (string), `customerId` (string), `endTime` (string), `eventName` (string), `filters` (string)
    Example: `gws admin-reports activities watch --params '{"applicationName":"...","userKey":"..."}'`

### channels

  - `stop` — Stop watching resources through this channel.

### customerUsageReports

  - `get` — Retrieves a report which is a collection of properties and statistics for a specific customer's account. For more information, see the Customers Usage Report guide. For more information about the customer report's parameters, see the Customers Usage parameters reference guides.
    Required: `date` (string)
    Key params: `customerId` (string), `pageToken` (string), `parameters` (string)
    Example: `gws admin-reports customerUsageReports get --params '{"date":"..."}'`

### entityUsageReports

  - `get` — Retrieves a report which is a collection of properties and statistics for entities used by users within the account. For more information, see the Entities Usage Report guide. For more information about the entities report's parameters, see the Entities Usage parameters reference guides.
    Required: `date` (string), `entityKey` (string), `entityType` (string)
    Key params: `customerId` (string), `filters` (string), `maxResults` (uint32, default: "1000"), `pageToken` (string), `parameters` (string)
    Example: `gws admin-reports entityUsageReports get --params '{"date":"...","entityKey":"...","entityType":"..."}'`

### userUsageReport

  - `get` — Retrieves a report which is a collection of properties and statistics for a set of users with the account. For more information, see the User Usage Report guide. For more information about the user report's parameters, see the Users Usage parameters reference guides.
    Required: `date` (string), `userKey` (string)
    Key params: `customerId` (string), `filters` (string), `groupIdFilter` (string), `maxResults` (uint32, default: "1000"), `orgUnitID` (string, default: "")
    Example: `gws admin-reports userUsageReport get --params '{"date":"...","userKey":"..."}'`

## Discovering Commands

Before calling any API method, inspect it:

```bash
# Browse resources and methods
gws admin-reports --help

# Inspect a method's required params, types, and defaults
gws schema admin-reports.<resource>.<method>
```

Use `gws schema` output to build your `--params` and `--json` flags.

