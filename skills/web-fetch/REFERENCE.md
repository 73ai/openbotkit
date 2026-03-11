## Usage

```bash
obk websearch fetch "url" [flags]
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | markdown | Output format: markdown, text |
| `--max-length` | | 100000 | Maximum content length in characters |
| `--no-cache` | | false | Bypass result cache |

## Output

JSON to stdout:

```json
{
  "url": "https://example.com",
  "title": "Example Domain",
  "content": "# Example Domain\n\nThis domain is for use in illustrative examples...",
  "content_type": "text/html; charset=UTF-8",
  "status_code": 200,
  "truncated": false,
  "cached": false
}
```

## Features

- **HTML to Markdown**: Automatically converts HTML pages to readable markdown
- **HTML to Text**: Use `--format text` for plain text extraction
- **JSON Pretty-Print**: JSON responses are automatically formatted
- **GitHub URL normalization**: `github.com/.../blob/...` URLs are converted to raw content URLs
- **SSRF Protection**: Blocks requests to private/loopback IP addresses
- **Content Truncation**: Long pages are truncated at `--max-length` with a marker
- **Caching**: Fetched pages are cached for 15 minutes. Use `--no-cache` to bypass
