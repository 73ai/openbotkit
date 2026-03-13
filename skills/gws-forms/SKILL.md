---
name: gws-forms
version: 1.0.0
description: "Read and write Google Forms."
metadata:
  openclaw:
    category: "productivity"
    requires:
      bins: ["gws"]
    cliHelp: "gws forms --help"
---

# forms (v1)

> **PREREQUISITE:** Read `../gws-shared/SKILL.md` for auth, global flags, and security rules. If missing, run `gws generate-skills` to create it.

```bash
gws forms <resource> <method> [flags]
```

## API Resources

### forms

  - `batchUpdate` — Change the form with a batch of updates.
    Required: `formId` (string)
    Example: `gws forms forms batchUpdate --params '{"formId":"..."}'`
  - `create` — Create a new form using the title given in the provided form message in the request. *Important:* Only the form.info.title and form.info.document_title fields are copied to the new form. All other fields including the form description, items and settings are disallowed. To create a new form and add items, you must first call forms.create to create an empty form with a title and (optional) document title, and then call forms.update to add the items.
    Key params: `unpublished` (boolean)
    Example: `gws forms forms create --params '{"unpublished":"..."}'`
  - `get` — Get a form.
    Required: `formId` (string)
    Example: `gws forms forms get --params '{"formId":"..."}'`
  - `setPublishSettings` — Updates the publish settings of a form. Legacy forms aren't supported because they don't have the `publish_settings` field.
    Required: `formId` (string)
    Example: `gws forms forms setPublishSettings --params '{"formId":"..."}'`
  - `responses` — Operations on the 'responses' resource
  - `watches` — Operations on the 'watches' resource

## Discovering Commands

Before calling any API method, inspect it:

```bash
# Browse resources and methods
gws forms --help

# Inspect a method's required params, types, and defaults
gws schema forms.<resource>.<method>
```

Use `gws schema` output to build your `--params` and `--json` flags.

