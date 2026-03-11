## Usage

```bash
obk websearch search "query" [flags]
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max-results` | `-n` | 10 | Maximum number of results |
| `--backend` | `-b` | auto | Search backend: auto, duckduckgo, brave, mojeek, yahoo, yandex, google, wikipedia |
| `--time-limit` | `-t` | | Time limit: d (day), w (week), m (month) |
| `--region` | `-r` | us-en | Region for search results |
| `--no-cache` | | false | Bypass result cache |

## Auto Backend Set

When `--backend auto` (default), searches use: DuckDuckGo + Brave + Mojeek + Wikipedia.

Yahoo, Yandex, and Google are opt-in only via `--backend <name>`.

## Output

JSON to stdout:

```json
{
  "query": "golang generics",
  "results": [
    {
      "title": "Tutorial: Getting started with generics",
      "url": "https://go.dev/doc/tutorial/generics",
      "snippet": "This tutorial introduces the basics of generics in Go.",
      "source": "duckduckgo"
    }
  ],
  "metadata": {
    "backends": ["wikipedia", "duckduckgo"],
    "search_time_ms": 450,
    "total_results": 5,
    "cached": false
  }
}
```

## Caching

Results are cached for 15 minutes by default. Repeated queries return cached results with `"cached": true` in metadata. Use `--no-cache` to bypass.

## Search History

Query past searches via `obk db websearch`:

```bash
# Recent searches
obk db websearch "SELECT query, result_count, backends, search_ms, created_at FROM search_history ORDER BY created_at DESC LIMIT 10;"

# Search frequency
obk db websearch "SELECT query, COUNT(*) as times FROM search_history GROUP BY query ORDER BY times DESC LIMIT 10;"
```
