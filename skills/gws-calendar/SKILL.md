---
name: gws-calendar
version: 1.0.0
description: "Google Calendar: Manage calendars and events."
metadata:
  openclaw:
    category: "productivity"
    requires:
      bins: ["gws"]
    cliHelp: "gws calendar --help"
---

# calendar (v3)

> **PREREQUISITE:** Read `../gws-shared/SKILL.md` for auth, global flags, and security rules. If missing, run `gws generate-skills` to create it.

```bash
gws calendar <resource> <method> [flags]
```

## Helper Commands

| Command | Description |
|---------|-------------|
| [`+insert`](../gws-calendar-insert/SKILL.md) | create a new event |
| [`+agenda`](../gws-calendar-agenda/SKILL.md) | Show upcoming events across all calendars |

## API Resources

### acl

  - `delete` — Deletes an access control rule.
    Required: `calendarId` (string), `ruleId` (string)
    Example: `gws calendar acl delete --params '{"calendarId":"...","ruleId":"..."}'`
  - `get` — Returns an access control rule.
    Required: `calendarId` (string), `ruleId` (string)
    Example: `gws calendar acl get --params '{"calendarId":"...","ruleId":"..."}'`
  - `insert` — Creates an access control rule.
    Required: `calendarId` (string)
    Key params: `sendNotifications` (boolean)
    Example: `gws calendar acl insert --params '{"calendarId":"..."}'`
  - `list` — Returns the rules in the access control list for the calendar.
    Required: `calendarId` (string)
    Key params: `maxResults` (int32), `pageToken` (string), `showDeleted` (boolean), `syncToken` (string)
    Example: `gws calendar acl list --params '{"calendarId":"..."}'`
  - `patch` — Updates an access control rule. This method supports patch semantics.
    Required: `calendarId` (string), `ruleId` (string)
    Key params: `sendNotifications` (boolean)
    Example: `gws calendar acl patch --params '{"calendarId":"...","ruleId":"..."}'`
  - `update` — Updates an access control rule.
    Required: `calendarId` (string), `ruleId` (string)
    Key params: `sendNotifications` (boolean)
    Example: `gws calendar acl update --params '{"calendarId":"...","ruleId":"..."}'`
  - `watch` — Watch for changes to ACL resources.
    Required: `calendarId` (string)
    Key params: `maxResults` (int32), `pageToken` (string), `showDeleted` (boolean), `syncToken` (string)
    Example: `gws calendar acl watch --params '{"calendarId":"..."}'`

### calendarList

  - `delete` — Removes a calendar from the user's calendar list.
    Required: `calendarId` (string)
    Example: `gws calendar calendarList delete --params '{"calendarId":"..."}'`
  - `get` — Returns a calendar from the user's calendar list.
    Required: `calendarId` (string)
    Example: `gws calendar calendarList get --params '{"calendarId":"..."}'`
  - `insert` — Inserts an existing calendar into the user's calendar list.
    Key params: `colorRgbFormat` (boolean)
    Example: `gws calendar calendarList insert --params '{"colorRgbFormat":"..."}'`
  - `list` — Returns the calendars on the user's calendar list.
    Key params: `maxResults` (int32), `minAccessRole` (string), `pageToken` (string), `showDeleted` (boolean), `showHidden` (boolean)
    Example: `gws calendar calendarList list --params '{"maxResults":"...","minAccessRole":"..."}'`
  - `patch` — Updates an existing calendar on the user's calendar list. This method supports patch semantics.
    Required: `calendarId` (string)
    Key params: `colorRgbFormat` (boolean)
    Example: `gws calendar calendarList patch --params '{"calendarId":"..."}'`
  - `update` — Updates an existing calendar on the user's calendar list.
    Required: `calendarId` (string)
    Key params: `colorRgbFormat` (boolean)
    Example: `gws calendar calendarList update --params '{"calendarId":"..."}'`
  - `watch` — Watch for changes to CalendarList resources.
    Key params: `maxResults` (int32), `minAccessRole` (string), `pageToken` (string), `showDeleted` (boolean), `showHidden` (boolean)
    Example: `gws calendar calendarList watch --params '{"maxResults":"...","minAccessRole":"..."}'`

### calendars

  - `clear` — Clears a primary calendar. This operation deletes all events associated with the primary calendar of an account.
    Required: `calendarId` (string)
    Example: `gws calendar calendars clear --params '{"calendarId":"..."}'`
  - `delete` — Deletes a secondary calendar. Use calendars.clear for clearing all events on primary calendars.
    Required: `calendarId` (string)
    Example: `gws calendar calendars delete --params '{"calendarId":"..."}'`
  - `get` — Returns metadata for a calendar.
    Required: `calendarId` (string)
    Example: `gws calendar calendars get --params '{"calendarId":"..."}'`
  - `insert` — Creates a secondary calendar.
The authenticated user for the request is made the data owner of the new calendar.

Note: We recommend to authenticate as the intended data owner of the calendar. You can use domain-wide delegation of authority to allow applications to act on behalf of a specific user. Don't use a service account for authentication. If you use a service account for authentication, the service account is the data owner, which can lead to unexpected behavior.
  - `patch` — Updates metadata for a calendar. This method supports patch semantics.
    Required: `calendarId` (string)
    Example: `gws calendar calendars patch --params '{"calendarId":"..."}'`
  - `update` — Updates metadata for a calendar.
    Required: `calendarId` (string)
    Example: `gws calendar calendars update --params '{"calendarId":"..."}'`

### channels

  - `stop` — Stop watching resources through this channel

### colors

  - `get` — Returns the color definitions for calendars and events.

### events

  - `delete` — Deletes an event.
    Required: `calendarId` (string), `eventId` (string)
    Key params: `sendNotifications` (boolean), `sendUpdates` (string)
    Example: `gws calendar events delete --params '{"calendarId":"...","eventId":"..."}'`
  - `get` — Returns an event based on its Google Calendar ID. To retrieve an event using its iCalendar ID, call the events.list method using the iCalUID parameter.
    Required: `calendarId` (string), `eventId` (string)
    Key params: `alwaysIncludeEmail` (boolean), `maxAttendees` (int32), `timeZone` (string)
    Example: `gws calendar events get --params '{"calendarId":"...","eventId":"..."}'`
  - `import` — Imports an event. This operation is used to add a private copy of an existing event to a calendar. Only events with an eventType of default may be imported.
Deprecated behavior: If a non-default event is imported, its type will be changed to default and any event-type-specific properties it may have will be dropped.
    Required: `calendarId` (string)
    Key params: `conferenceDataVersion` (int32), `supportsAttachments` (boolean)
    Example: `gws calendar events import --params '{"calendarId":"..."}'`
  - `insert` — Creates an event.
    Required: `calendarId` (string)
    Key params: `conferenceDataVersion` (int32), `maxAttendees` (int32), `sendNotifications` (boolean), `sendUpdates` (string), `supportsAttachments` (boolean)
    Example: `gws calendar events insert --params '{"calendarId":"..."}'`
  - `instances` — Returns instances of the specified recurring event.
    Required: `calendarId` (string), `eventId` (string)
    Key params: `alwaysIncludeEmail` (boolean), `maxAttendees` (int32), `maxResults` (int32), `originalStart` (string), `pageToken` (string)
    Example: `gws calendar events instances --params '{"calendarId":"...","eventId":"..."}'`
  - `list` — Returns events on the specified calendar.
    Required: `calendarId` (string)
    Key params: `alwaysIncludeEmail` (boolean), `eventTypes` (string), `iCalUID` (string), `maxAttendees` (int32), `maxResults` (int32, default: "250")
    Example: `gws calendar events list --params '{"calendarId":"..."}'`
  - `move` — Moves an event to another calendar, i.e. changes an event's organizer. Note that only default events can be moved; birthday, focusTime, fromGmail, outOfOffice and workingLocation events cannot be moved.
    Required: `calendarId` (string), `destination` (string), `eventId` (string)
    Key params: `sendNotifications` (boolean), `sendUpdates` (string)
    Example: `gws calendar events move --params '{"calendarId":"...","destination":"...","eventId":"..."}'`
  - `patch` — Updates an event. This method supports patch semantics.
    Required: `calendarId` (string), `eventId` (string)
    Key params: `alwaysIncludeEmail` (boolean), `conferenceDataVersion` (int32), `maxAttendees` (int32), `sendNotifications` (boolean), `sendUpdates` (string)
    Example: `gws calendar events patch --params '{"calendarId":"...","eventId":"..."}'`
  - `quickAdd` — Creates an event based on a simple text string.
    Required: `calendarId` (string), `text` (string)
    Key params: `sendNotifications` (boolean), `sendUpdates` (string)
    Example: `gws calendar events quickAdd --params '{"calendarId":"...","text":"..."}'`
  - `update` — Updates an event.
    Required: `calendarId` (string), `eventId` (string)
    Key params: `alwaysIncludeEmail` (boolean), `conferenceDataVersion` (int32), `maxAttendees` (int32), `sendNotifications` (boolean), `sendUpdates` (string)
    Example: `gws calendar events update --params '{"calendarId":"...","eventId":"..."}'`
  - `watch` — Watch for changes to Events resources.
    Required: `calendarId` (string)
    Key params: `alwaysIncludeEmail` (boolean), `eventTypes` (string), `iCalUID` (string), `maxAttendees` (int32), `maxResults` (int32, default: "250")
    Example: `gws calendar events watch --params '{"calendarId":"..."}'`

### freebusy

  - `query` — Returns free/busy information for a set of calendars.

### settings

  - `get` — Returns a single user setting.
    Required: `setting` (string)
    Example: `gws calendar settings get --params '{"setting":"..."}'`
  - `list` — Returns all user settings for the authenticated user.
    Key params: `maxResults` (int32), `pageToken` (string), `syncToken` (string)
    Example: `gws calendar settings list --params '{"maxResults":"...","pageToken":"..."}'`
  - `watch` — Watch for changes to Settings resources.
    Key params: `maxResults` (int32), `pageToken` (string), `syncToken` (string)
    Example: `gws calendar settings watch --params '{"maxResults":"...","pageToken":"..."}'`

## Discovering Commands

Before calling any API method, inspect it:

```bash
# Browse resources and methods
gws calendar --help

# Inspect a method's required params, types, and defaults
gws schema calendar.<resource>.<method>
```

Use `gws schema` output to build your `--params` and `--json` flags.

