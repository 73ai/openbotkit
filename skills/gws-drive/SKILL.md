---
name: gws-drive
version: 1.0.0
description: "Google Drive: Manage files, folders, and shared drives."
metadata:
  openclaw:
    category: "productivity"
    requires:
      bins: ["gws"]
    cliHelp: "gws drive --help"
---

# drive (v3)

> **PREREQUISITE:** Read `../gws-shared/SKILL.md` for auth, global flags, and security rules. If missing, run `gws generate-skills` to create it.

```bash
gws drive <resource> <method> [flags]
```

## Helper Commands

| Command | Description |
|---------|-------------|
| [`+upload`](../gws-drive-upload/SKILL.md) | Upload a file with automatic metadata |

## API Resources

### about

  - `get` — Gets information about the user, the user's Drive, and system capabilities. For more information, see [Return user info](https://developers.google.com/workspace/drive/api/guides/user-info). Required: The `fields` parameter must be set. To return the exact fields you need, see [Return specific fields](https://developers.google.com/workspace/drive/api/guides/fields-parameter).

### accessproposals

  - `get` — Retrieves an access proposal by ID. For more information, see [Manage pending access proposals](https://developers.google.com/workspace/drive/api/guides/pending-access).
    Required: `fileId` (string), `proposalId` (string)
    Example: `gws drive accessproposals get --params '{"fileId":"...","proposalId":"..."}'`
  - `list` — List the access proposals on a file. For more information, see [Manage pending access proposals](https://developers.google.com/workspace/drive/api/guides/pending-access). Note: Only approvers are able to list access proposals on a file. If the user isn't an approver, a 403 error is returned.
    Required: `fileId` (string)
    Key params: `pageSize` (int32), `pageToken` (string)
    Example: `gws drive accessproposals list --params '{"fileId":"..."}'`
  - `resolve` — Approves or denies an access proposal. For more information, see [Manage pending access proposals](https://developers.google.com/workspace/drive/api/guides/pending-access).
    Required: `fileId` (string), `proposalId` (string)
    Example: `gws drive accessproposals resolve --params '{"fileId":"...","proposalId":"..."}'`

### approvals

  - `get` — Gets an Approval by ID.
    Required: `approvalId` (string), `fileId` (string)
    Example: `gws drive approvals get --params '{"approvalId":"...","fileId":"..."}'`
  - `list` — Lists the Approvals on a file.
    Required: `fileId` (string)
    Key params: `pageSize` (int32), `pageToken` (string)
    Example: `gws drive approvals list --params '{"fileId":"..."}'`

### apps

  - `get` — Gets a specific app. For more information, see [Return user info](https://developers.google.com/workspace/drive/api/guides/user-info).
    Required: `appId` (string)
    Example: `gws drive apps get --params '{"appId":"..."}'`
  - `list` — Lists a user's installed apps. For more information, see [Return user info](https://developers.google.com/workspace/drive/api/guides/user-info).
    Key params: `appFilterExtensions` (string, default: ""), `appFilterMimeTypes` (string, default: ""), `languageCode` (string)
    Example: `gws drive apps list --params '{"appFilterExtensions":"","appFilterMimeTypes":""}'`

### changes

  - `getStartPageToken` — Gets the starting pageToken for listing future changes. For more information, see [Retrieve changes](https://developers.google.com/workspace/drive/api/guides/manage-changes).
    Key params: `driveId` (string), `supportsAllDrives` (boolean, default: "false")
    Example: `gws drive changes getStartPageToken --params '{"driveId":"...","supportsAllDrives":"false"}'`
  - `list` — Lists the changes for a user or shared drive. For more information, see [Retrieve changes](https://developers.google.com/workspace/drive/api/guides/manage-changes).
    Required: `pageToken` (string)
    Key params: `driveId` (string), `includeCorpusRemovals` (boolean, default: "false"), `includeItemsFromAllDrives` (boolean, default: "false"), `includeLabels` (string), `includePermissionsForView` (string)
    Example: `gws drive changes list --params '{"pageToken":"..."}'`
  - `watch` — Subscribes to changes for a user. For more information, see [Notifications for resource changes](https://developers.google.com/workspace/drive/api/guides/push).
    Required: `pageToken` (string)
    Key params: `driveId` (string), `includeCorpusRemovals` (boolean, default: "false"), `includeItemsFromAllDrives` (boolean, default: "false"), `includeLabels` (string), `includePermissionsForView` (string)
    Example: `gws drive changes watch --params '{"pageToken":"..."}'`

### channels

  - `stop` — Stops watching resources through this channel. For more information, see [Notifications for resource changes](https://developers.google.com/workspace/drive/api/guides/push).

### comments

  - `create` — Creates a comment on a file. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments). Required: The `fields` parameter must be set. To return the exact fields you need, see [Return specific fields](https://developers.google.com/workspace/drive/api/guides/fields-parameter).
    Required: `fileId` (string)
    Example: `gws drive comments create --params '{"fileId":"..."}'`
  - `delete` — Deletes a comment. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments).
    Required: `commentId` (string), `fileId` (string)
    Example: `gws drive comments delete --params '{"commentId":"...","fileId":"..."}'`
  - `get` — Gets a comment by ID. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments). Required: The `fields` parameter must be set. To return the exact fields you need, see [Return specific fields](https://developers.google.com/workspace/drive/api/guides/fields-parameter).
    Required: `commentId` (string), `fileId` (string)
    Key params: `includeDeleted` (boolean, default: "false")
    Example: `gws drive comments get --params '{"commentId":"...","fileId":"..."}'`
  - `list` — Lists a file's comments. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments). Required: The `fields` parameter must be set. To return the exact fields you need, see [Return specific fields](https://developers.google.com/workspace/drive/api/guides/fields-parameter).
    Required: `fileId` (string)
    Key params: `includeDeleted` (boolean, default: "false"), `pageSize` (int32, default: "20"), `pageToken` (string), `startModifiedTime` (string)
    Example: `gws drive comments list --params '{"fileId":"..."}'`
  - `update` — Updates a comment with patch semantics. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments). Required: The `fields` parameter must be set. To return the exact fields you need, see [Return specific fields](https://developers.google.com/workspace/drive/api/guides/fields-parameter).
    Required: `commentId` (string), `fileId` (string)
    Example: `gws drive comments update --params '{"commentId":"...","fileId":"..."}'`

### drives

  - `create` — Creates a shared drive. For more information, see [Manage shared drives](https://developers.google.com/workspace/drive/api/guides/manage-shareddrives).
    Required: `requestId` (string)
    Example: `gws drive drives create --params '{"requestId":"..."}'`
  - `get` — Gets a shared drive's metadata by ID. For more information, see [Manage shared drives](https://developers.google.com/workspace/drive/api/guides/manage-shareddrives).
    Required: `driveId` (string)
    Key params: `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive drives get --params '{"driveId":"..."}'`
  - `hide` — Hides a shared drive from the default view. For more information, see [Manage shared drives](https://developers.google.com/workspace/drive/api/guides/manage-shareddrives).
    Required: `driveId` (string)
    Example: `gws drive drives hide --params '{"driveId":"..."}'`
  - `list` — Lists the user's shared drives. This method accepts the `q` parameter, which is a search query combining one or more search terms. For more information, see the [Search for shared drives](/workspace/drive/api/guides/search-shareddrives) guide.
    Key params: `pageSize` (int32, default: "10"), `pageToken` (string), `q` (string), `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive drives list --params '{"pageSize":"10","pageToken":"..."}'`
  - `unhide` — Restores a shared drive to the default view. For more information, see [Manage shared drives](https://developers.google.com/workspace/drive/api/guides/manage-shareddrives).
    Required: `driveId` (string)
    Example: `gws drive drives unhide --params '{"driveId":"..."}'`
  - `update` — Updates the metadata for a shared drive. For more information, see [Manage shared drives](https://developers.google.com/workspace/drive/api/guides/manage-shareddrives).
    Required: `driveId` (string)
    Key params: `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive drives update --params '{"driveId":"..."}'`

### files

  - `copy` — Creates a copy of a file and applies any requested updates with patch semantics. For more information, see [Create and manage files](https://developers.google.com/workspace/drive/api/guides/create-file).
    Required: `fileId` (string)
    Key params: `ignoreDefaultVisibility` (boolean, default: "false"), `includeLabels` (string), `includePermissionsForView` (string), `keepRevisionForever` (boolean, default: "false"), `ocrLanguage` (string)
    Example: `gws drive files copy --params '{"fileId":"..."}'`
  - `create` — Creates a file. For more information, see [Create and manage files](/workspace/drive/api/guides/create-file). This method supports an */upload* URI and accepts uploaded media with the following characteristics: - *Maximum file size:* 5,120 GB - *Accepted Media MIME types:* `*/*` (Specify a valid MIME type, rather than the literal `*/*` value. The literal `*/*` is only used to indicate that any valid MIME type can be uploaded.
    Key params: `ignoreDefaultVisibility` (boolean, default: "false"), `includeLabels` (string), `includePermissionsForView` (string), `keepRevisionForever` (boolean, default: "false"), `ocrLanguage` (string)
    Example: `gws drive files create --params '{"ignoreDefaultVisibility":"false","includeLabels":"..."}'`
  - `download` — Downloads the content of a file. For more information, see [Download and export files](https://developers.google.com/workspace/drive/api/guides/manage-downloads). Operations are valid for 24 hours from the time of creation.
    Required: `fileId` (string)
    Key params: `mimeType` (string), `revisionId` (string)
    Example: `gws drive files download --params '{"fileId":"..."}'`
  - `export` — Exports a Google Workspace document to the requested MIME type and returns exported byte content. For more information, see [Download and export files](https://developers.google.com/workspace/drive/api/guides/manage-downloads). Note that the exported content is limited to 10 MB.
    Required: `fileId` (string), `mimeType` (string)
    Example: `gws drive files export --params '{"fileId":"...","mimeType":"..."}'`
  - `generateIds` — Generates a set of file IDs which can be provided in create or copy requests. For more information, see [Create and manage files](https://developers.google.com/workspace/drive/api/guides/create-file).
    Key params: `count` (int32, default: "10"), `space` (string, default: "drive"), `type` (string, default: "files")
    Example: `gws drive files generateIds --params '{"count":"10","space":"drive"}'`
  - `get` — Gets a file's metadata or content by ID. For more information, see [Search for files and folders](/workspace/drive/api/guides/search-files). If you provide the URL parameter `alt=media`, then the response includes the file contents in the response body. Downloading content with `alt=media` only works if the file is stored in Drive. To download Google Docs, Sheets, and Slides use [`files.export`](/workspace/drive/api/reference/rest/v3/files/export) instead.
    Required: `fileId` (string)
    Key params: `acknowledgeAbuse` (boolean, default: "false"), `includeLabels` (string), `includePermissionsForView` (string), `supportsAllDrives` (boolean, default: "false")
    Example: `gws drive files get --params '{"fileId":"..."}'`
  - `list` — Lists the user's files. For more information, see [Search for files and folders](/workspace/drive/api/guides/search-files). This method accepts the `q` parameter, which is a search query combining one or more search terms. This method returns *all* files by default, including trashed files. If you don't want trashed files to appear in the list, use the `trashed=false` query parameter to remove trashed files from the results.
    Key params: `corpora` (string), `driveId` (string), `includeItemsFromAllDrives` (boolean, default: "false"), `includeLabels` (string), `includePermissionsForView` (string)
    Example: `gws drive files list --params '{"corpora":"...","driveId":"..."}'`
  - `listLabels` — Lists the labels on a file. For more information, see [List labels on a file](https://developers.google.com/workspace/drive/api/guides/list-labels).
    Required: `fileId` (string)
    Key params: `maxResults` (int32, default: "100"), `pageToken` (string)
    Example: `gws drive files listLabels --params '{"fileId":"..."}'`
  - `modifyLabels` — Modifies the set of labels applied to a file. For more information, see [Set a label field on a file](https://developers.google.com/workspace/drive/api/guides/set-label). Returns a list of the labels that were added or modified.
    Required: `fileId` (string)
    Example: `gws drive files modifyLabels --params '{"fileId":"..."}'`
  - `update` — Updates a file's metadata, content, or both. When calling this method, only populate fields in the request that you want to modify. When updating fields, some fields might be changed automatically, such as `modifiedDate`. This method supports patch semantics. This method supports an */upload* URI and accepts uploaded media with the following characteristics: - *Maximum file size:* 5,120 GB - *Accepted Media MIME types:* `*/*` (Specify a valid MIME type, rather than the literal `*/*` value.
    Required: `fileId` (string)
    Key params: `addParents` (string), `includeLabels` (string), `includePermissionsForView` (string), `keepRevisionForever` (boolean, default: "false"), `ocrLanguage` (string)
    Example: `gws drive files update --params '{"fileId":"..."}'`
  - `watch` — Subscribes to changes to a file. For more information, see [Notifications for resource changes](https://developers.google.com/workspace/drive/api/guides/push).
    Required: `fileId` (string)
    Key params: `acknowledgeAbuse` (boolean, default: "false"), `includeLabels` (string), `includePermissionsForView` (string), `supportsAllDrives` (boolean, default: "false")
    Example: `gws drive files watch --params '{"fileId":"..."}'`

### operations

  - `get` — Gets the latest state of a long-running operation. Clients can use this method to poll the operation result at intervals as recommended by the API service.
    Required: `name` (string)
    Example: `gws drive operations get --params '{"name":"..."}'`

### permissions

  - `create` — Creates a permission for a file or shared drive. For more information, see [Share files, folders, and drives](https://developers.google.com/workspace/drive/api/guides/manage-sharing). **Warning:** Concurrent permissions operations on the same file aren't supported; only the last update is applied.
    Required: `fileId` (string)
    Key params: `emailMessage` (string), `moveToNewOwnersRoot` (boolean, default: "false"), `sendNotificationEmail` (boolean), `supportsAllDrives` (boolean, default: "false"), `transferOwnership` (boolean, default: "false")
    Example: `gws drive permissions create --params '{"fileId":"..."}'`
  - `delete` — Deletes a permission. For more information, see [Share files, folders, and drives](https://developers.google.com/workspace/drive/api/guides/manage-sharing). **Warning:** Concurrent permissions operations on the same file aren't supported; only the last update is applied.
    Required: `fileId` (string), `permissionId` (string)
    Key params: `supportsAllDrives` (boolean, default: "false"), `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive permissions delete --params '{"fileId":"...","permissionId":"..."}'`
  - `get` — Gets a permission by ID. For more information, see [Share files, folders, and drives](https://developers.google.com/workspace/drive/api/guides/manage-sharing).
    Required: `fileId` (string), `permissionId` (string)
    Key params: `supportsAllDrives` (boolean, default: "false"), `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive permissions get --params '{"fileId":"...","permissionId":"..."}'`
  - `list` — Lists a file's or shared drive's permissions. For more information, see [Share files, folders, and drives](https://developers.google.com/workspace/drive/api/guides/manage-sharing).
    Required: `fileId` (string)
    Key params: `includePermissionsForView` (string), `pageSize` (int32), `pageToken` (string), `supportsAllDrives` (boolean, default: "false"), `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive permissions list --params '{"fileId":"..."}'`
  - `update` — Updates a permission with patch semantics. For more information, see [Share files, folders, and drives](https://developers.google.com/workspace/drive/api/guides/manage-sharing). **Warning:** Concurrent permissions operations on the same file aren't supported; only the last update is applied.
    Required: `fileId` (string), `permissionId` (string)
    Key params: `removeExpiration` (boolean, default: "false"), `supportsAllDrives` (boolean, default: "false"), `transferOwnership` (boolean, default: "false"), `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive permissions update --params '{"fileId":"...","permissionId":"..."}'`

### replies

  - `create` — Creates a reply to a comment. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments).
    Required: `commentId` (string), `fileId` (string)
    Example: `gws drive replies create --params '{"commentId":"...","fileId":"..."}'`
  - `delete` — Deletes a reply. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments).
    Required: `commentId` (string), `fileId` (string), `replyId` (string)
    Example: `gws drive replies delete --params '{"commentId":"...","fileId":"...","replyId":"..."}'`
  - `get` — Gets a reply by ID. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments).
    Required: `commentId` (string), `fileId` (string), `replyId` (string)
    Key params: `includeDeleted` (boolean, default: "false")
    Example: `gws drive replies get --params '{"commentId":"...","fileId":"...","replyId":"..."}'`
  - `list` — Lists a comment's replies. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments).
    Required: `commentId` (string), `fileId` (string)
    Key params: `includeDeleted` (boolean, default: "false"), `pageSize` (int32, default: "20"), `pageToken` (string)
    Example: `gws drive replies list --params '{"commentId":"...","fileId":"..."}'`
  - `update` — Updates a reply with patch semantics. For more information, see [Manage comments and replies](https://developers.google.com/workspace/drive/api/guides/manage-comments).
    Required: `commentId` (string), `fileId` (string), `replyId` (string)
    Example: `gws drive replies update --params '{"commentId":"...","fileId":"...","replyId":"..."}'`

### revisions

  - `delete` — Permanently deletes a file version. You can only delete revisions for files with binary content in Google Drive, like images or videos. Revisions for other files, like Google Docs or Sheets, and the last remaining file version can't be deleted. For more information, see [Manage file revisions](https://developers.google.com/drive/api/guides/manage-revisions).
    Required: `fileId` (string), `revisionId` (string)
    Example: `gws drive revisions delete --params '{"fileId":"...","revisionId":"..."}'`
  - `get` — Gets a revision's metadata or content by ID. For more information, see [Manage file revisions](https://developers.google.com/workspace/drive/api/guides/manage-revisions).
    Required: `fileId` (string), `revisionId` (string)
    Key params: `acknowledgeAbuse` (boolean, default: "false")
    Example: `gws drive revisions get --params '{"fileId":"...","revisionId":"..."}'`
  - `list` — Lists a file's revisions. For more information, see [Manage file revisions](https://developers.google.com/workspace/drive/api/guides/manage-revisions). **Important:** The list of revisions returned by this method might be incomplete for files with a large revision history, including frequently edited Google Docs, Sheets, and Slides. Older revisions might be omitted from the response, meaning the first revision returned may not be the oldest existing revision.
    Required: `fileId` (string)
    Key params: `pageSize` (int32, default: "200"), `pageToken` (string)
    Example: `gws drive revisions list --params '{"fileId":"..."}'`
  - `update` — Updates a revision with patch semantics. For more information, see [Manage file revisions](https://developers.google.com/workspace/drive/api/guides/manage-revisions).
    Required: `fileId` (string), `revisionId` (string)
    Example: `gws drive revisions update --params '{"fileId":"...","revisionId":"..."}'`

### teamdrives

  - `create` — Deprecated: Use `drives.create` instead.
    Required: `requestId` (string)
    Example: `gws drive teamdrives create --params '{"requestId":"..."}'`
  - `get` — Deprecated: Use `drives.get` instead.
    Required: `teamDriveId` (string)
    Key params: `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive teamdrives get --params '{"teamDriveId":"..."}'`
  - `list` — Deprecated: Use `drives.list` instead.
    Key params: `pageSize` (int32, default: "10"), `pageToken` (string), `q` (string), `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive teamdrives list --params '{"pageSize":"10","pageToken":"..."}'`
  - `update` — Deprecated: Use `drives.update` instead.
    Required: `teamDriveId` (string)
    Key params: `useDomainAdminAccess` (boolean, default: "false")
    Example: `gws drive teamdrives update --params '{"teamDriveId":"..."}'`

## Discovering Commands

Before calling any API method, inspect it:

```bash
# Browse resources and methods
gws drive --help

# Inspect a method's required params, types, and defaults
gws schema drive.<resource>.<method>
```

Use `gws schema` output to build your `--params` and `--json` flags.

