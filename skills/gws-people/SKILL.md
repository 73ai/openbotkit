---
name: gws-people
version: 1.0.0
description: "Google People: Manage contacts and profiles."
metadata:
  openclaw:
    category: "productivity"
    requires:
      bins: ["gws"]
    cliHelp: "gws people --help"
---

# people (v1)

> **PREREQUISITE:** Read `../gws-shared/SKILL.md` for auth, global flags, and security rules. If missing, run `gws generate-skills` to create it.

```bash
gws people <resource> <method> [flags]
```

## API Resources

### contactGroups

  - `batchGet` — Get a list of contact groups owned by the authenticated user by specifying a list of contact group resource names.
    Key params: `groupFields` (google-fieldmask), `maxMembers` (int32), `resourceNames` (string)
    Example: `gws people contactGroups batchGet --params '{"groupFields":"...","maxMembers":"..."}'`
  - `create` — Create a new contact group owned by the authenticated user. Created contact group names must be unique to the users contact groups. Attempting to create a group with a duplicate name will return a HTTP 409 error. Mutate requests for the same user should be sent sequentially to avoid increased latency and failures.
  - `delete` — Delete an existing contact group owned by the authenticated user by specifying a contact group resource name. Mutate requests for the same user should be sent sequentially to avoid increased latency and failures.
    Required: `resourceName` (string)
    Key params: `deleteContacts` (boolean)
    Example: `gws people contactGroups delete --params '{"resourceName":"..."}'`
  - `get` — Get a specific contact group owned by the authenticated user by specifying a contact group resource name.
    Required: `resourceName` (string)
    Key params: `groupFields` (google-fieldmask), `maxMembers` (int32)
    Example: `gws people contactGroups get --params '{"resourceName":"..."}'`
  - `list` — List all contact groups owned by the authenticated user. Members of the contact groups are not populated.
    Key params: `groupFields` (google-fieldmask), `pageSize` (int32), `pageToken` (string), `syncToken` (string)
    Example: `gws people contactGroups list --params '{"groupFields":"...","pageSize":"..."}'`
  - `update` — Update the name of an existing contact group owned by the authenticated user. Updated contact group names must be unique to the users contact groups. Attempting to create a group with a duplicate name will return a HTTP 409 error. Mutate requests for the same user should be sent sequentially to avoid increased latency and failures.
    Required: `resourceName` (string)
    Example: `gws people contactGroups update --params '{"resourceName":"..."}'`
  - `members` — Operations on the 'members' resource

### otherContacts

  - `copyOtherContactToMyContactsGroup` — Copies an "Other contact" to a new contact in the user's "myContacts" group Mutate requests for the same user should be sent sequentially to avoid increased latency and failures.
    Required: `resourceName` (string)
    Example: `gws people otherContacts copyOtherContactToMyContactsGroup --params '{"resourceName":"..."}'`
  - `list` — List all "Other contacts", that is contacts that are not in a contact group. "Other contacts" are typically auto created contacts from interactions. Sync tokens expire 7 days after the full sync. A request with an expired sync token will get an error with an [google.rpc.ErrorInfo](https://cloud.google.com/apis/design/errors#error_info) with reason "EXPIRED_SYNC_TOKEN". In the case of such an error clients should make a full sync request without a `sync_token`.
    Key params: `pageSize` (int32), `pageToken` (string), `readMask` (google-fieldmask), `requestSyncToken` (boolean), `sources` (string)
    Example: `gws people otherContacts list --params '{"pageSize":"...","pageToken":"..."}'`
  - `search` — Provides a list of contacts in the authenticated user's other contacts that matches the search query. The query matches on a contact's `names`, `emailAddresses`, and `phoneNumbers` fields that are from the OTHER_CONTACT source. **IMPORTANT**: Before searching, clients should send a warmup request with an empty query to update the cache. See https://developers.google.com/people/v1/other-contacts#search_the_users_other_contacts
    Key params: `pageSize` (int32), `query` (string), `readMask` (google-fieldmask)
    Example: `gws people otherContacts search --params '{"pageSize":"...","query":"..."}'`

### people

  - `batchCreateContacts` — Create a batch of new contacts and return the PersonResponses for the newly Mutate requests for the same user should be sent sequentially to avoid increased latency and failures.
  - `batchUpdateContacts` — Update a batch of contacts and return a map of resource names to PersonResponses for the updated contacts. Mutate requests for the same user should be sent sequentially to avoid increased latency and failures.
  - `createContact` — Create a new contact and return the person resource for that contact. The request returns a 400 error if more than one field is specified on a field that is a singleton for contact sources: * biographies * birthdays * genders * names Mutate requests for the same user should be sent sequentially to avoid increased latency and failures.
    Key params: `personFields` (google-fieldmask), `sources` (string)
    Example: `gws people people createContact --params '{"personFields":"...","sources":"..."}'`
  - `deleteContactPhoto` — Delete a contact's photo. Mutate requests for the same user should be done sequentially to avoid // lock contention.
    Required: `resourceName` (string)
    Key params: `personFields` (google-fieldmask), `sources` (string)
    Example: `gws people people deleteContactPhoto --params '{"resourceName":"..."}'`
  - `get` — Provides information about a person by specifying a resource name. Use `people/me` to indicate the authenticated user. The request returns a 400 error if 'personFields' is not specified.
    Required: `resourceName` (string)
    Key params: `personFields` (google-fieldmask), `requestMask.includeField` (google-fieldmask), `sources` (string)
    Example: `gws people people get --params '{"resourceName":"..."}'`
  - `getBatchGet` — Provides information about a list of specific people by specifying a list of requested resource names. Use `people/me` to indicate the authenticated user. The request returns a 400 error if 'personFields' is not specified.
    Key params: `personFields` (google-fieldmask), `requestMask.includeField` (google-fieldmask), `resourceNames` (string), `sources` (string)
    Example: `gws people people getBatchGet --params '{"personFields":"...","requestMask.includeField":"..."}'`
  - `listDirectoryPeople` — Provides a list of domain profiles and domain contacts in the authenticated user's domain directory. When the `sync_token` is specified, resources deleted since the last sync will be returned as a person with `PersonMetadata.deleted` set to true. When the `page_token` or `sync_token` is specified, all other request parameters must match the first call. Writes may have a propagation delay of several minutes for sync requests. Incremental syncs are not intended for read-after-write use cases.
    Key params: `mergeSources` (string), `pageSize` (int32), `pageToken` (string), `readMask` (google-fieldmask), `requestSyncToken` (boolean)
    Example: `gws people people listDirectoryPeople --params '{"mergeSources":"...","pageSize":"..."}'`
  - `searchContacts` — Provides a list of contacts in the authenticated user's grouped contacts that matches the search query. The query matches on a contact's `names`, `nickNames`, `emailAddresses`, `phoneNumbers`, and `organizations` fields that are from the CONTACT source. **IMPORTANT**: Before searching, clients should send a warmup request with an empty query to update the cache. See https://developers.google.com/people/v1/contacts#search_the_users_contacts
    Key params: `pageSize` (int32), `query` (string), `readMask` (google-fieldmask), `sources` (string)
    Example: `gws people people searchContacts --params '{"pageSize":"...","query":"..."}'`
  - `searchDirectoryPeople` — Provides a list of domain profiles and domain contacts in the authenticated user's domain directory that match the search query.
    Key params: `mergeSources` (string), `pageSize` (int32), `pageToken` (string), `query` (string), `readMask` (google-fieldmask)
    Example: `gws people people searchDirectoryPeople --params '{"mergeSources":"...","pageSize":"..."}'`
  - `updateContact` — Update contact data for an existing contact person. Any non-contact data will not be modified. Any non-contact data in the person to update will be ignored. All fields specified in the `update_mask` will be replaced. The server returns a 400 error if `person.metadata.sources` is not specified for the contact to be updated or if there is no contact source.
    Required: `resourceName` (string)
    Key params: `personFields` (google-fieldmask), `sources` (string), `updatePersonFields` (google-fieldmask)
    Example: `gws people people updateContact --params '{"resourceName":"..."}'`
  - `updateContactPhoto` — Update a contact's photo. Mutate requests for the same user should be sent sequentially to avoid increased latency and failures.
    Required: `resourceName` (string)
    Example: `gws people people updateContactPhoto --params '{"resourceName":"..."}'`
  - `connections` — Operations on the 'connections' resource

## Discovering Commands

Before calling any API method, inspect it:

```bash
# Browse resources and methods
gws people --help

# Inspect a method's required params, types, and defaults
gws schema people.<resource>.<method>
```

Use `gws schema` output to build your `--params` and `--json` flags.

