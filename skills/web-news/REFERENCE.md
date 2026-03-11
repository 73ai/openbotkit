## Usage

```bash
obk websearch news "query" [flags]
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max-results` | `-n` | 10 | Maximum number of results |
| `--backend` | `-b` | auto | News backend: auto, duckduckgo, yahoo |
| `--time-limit` | `-t` | | Time limit: d (day), w (week), m (month) |
| `--region` | `-r` | us-en | Region for news results |

## Output

JSON to stdout:

```json
{
  "query": "AI developments",
  "results": [
    {
      "title": "New AI Model Released",
      "url": "https://example.com/news/ai-model",
      "snippet": "A new AI model was released today...",
      "source": "duckduckgo",
      "date": "1700000000",
      "image": "https://img.example.com/thumbnail.jpg"
    }
  ],
  "metadata": {
    "backends": ["duckduckgo", "yahoo"],
    "search_time_ms": 650,
    "total_results": 8
  }
}
```

## Fields

- **date**: Unix timestamp (DuckDuckGo) or relative time like "2 hours ago" (Yahoo)
- **image**: Thumbnail URL when available, empty string otherwise
- **source**: Always the engine name ("duckduckgo" or "yahoo")

## Examples

```bash
# Latest AI news
obk websearch news "artificial intelligence"

# News from the past day only
obk websearch news "golang" --time-limit d

# Top 5 news from DuckDuckGo only
obk websearch news "climate" --backend duckduckgo --max-results 5
```
