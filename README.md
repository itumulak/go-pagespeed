# WordPress PageSpeed Analyzer

A Go-based tool that automatically analyzes all pages of a WordPress site using Google PageSpeed Insights API.

## Features

- 🚀 Automatically fetches all pages from any WordPress site
- 📊 Analyzes Performance, Accessibility, Best Practices, and SEO scores
- 📈 Captures Core Web Vitals (FID, FCP)
- 🔄 Automatic retry mechanism with exponential backoff
- ⚡ Built-in rate limiting to respect API quotas
- 💻 Real-time results display as they become available

## Prerequisites

- [Go](https://golang.org/dl/) 1.16 or higher
- Google PageSpeed Insights API key ([Get one here](https://developers.google.com/speed/docs/insights/v5/get-started))

### BASIC USAGE

```bash
go run . https://your-wordpress-site.com --key YOUR_API_KEY
```

### With Custom Rate Limit

```bash
go run . https://your-wordpress-site.com --key YOUR_API_KEY --rps 2
```

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--key` | (required) | Google PageSpeed Insights API key |
| `--rps` | 4 | Requests per second (rate limiting) |


### Examples

```bash
# Analyze with default settings (4 requests/second)
go run . https://example.com --key abc123xyz

# Slower rate for API quota conservation
go run . https://example.com --key abc123xyz --rps 2

# Analyze specific WordPress REST API endpoint
go run . https://example.com/wp-json/wp/v2/pages --key abc123xyz
```

### Output Example

```bash
🚀 WordPress PageSpeed Analyzer
================================
⚙️  Rate limit: 4 requests per second
Fetching WordPress pages...
✅ Found 25 pages
Analyzing with PageSpeed Insights (up to 3 retries per page)...

📄 About Us
   🔗 https://example.com/about-us
   🎯 Performance Score: 85/100
   ♿ Accessibility: 92/100
   ✨ Best Practices: 78/100
   🔍 SEO: 88/100
   📊 Core Web Vitals:
      • First Input Delay: 120 ms
      • First Contentful Paint: 1800 ms

================================
📊 Summary: Completed 25/25 pages in 2m15s
```

## How It Works

1. **Fetches WordPress Pages** - Uses WordPress REST API to get all pages
2. **Rate Limited Requests** - Controls API request frequency (default 4 req/sec)
3. **Concurrent Processing** - Multiple pages analyzed simultaneously
4. **Automatic Retries** - Failed requests retry up to 3 times with backoff
5. **Real-time Display** - Results shown immediately as they complete

### Rate Limiting
The tool respects Google's API quotas by implementing a token bucket rate limiter:

- Default: 4 requests per second
- Configurable via `--rps` flag
- Prevents API quota exhaustion

### Error Handling

- Automatic retry on network failures (max 3 attempts)
- Exponential backoff between retries
- Extended timeout for slow responses (60 seconds)
- Graceful handling of API errors

### Limitations

- WordPress site must have REST API enabled (default for WordPress 4.7+)
- Each page analysis counts toward your Google PageSpeed API quota
- Very large sites may take significant time to analyze

### API Quota Considerations

Google PageSpeed Insights API has quotas:

- Free tier: 25,000 queries per day
- Rate limits: 240 queries per minute

The tool's default 4 requests/second (240/minute) respects these limits.

### Troubleshooting

__"Error fetching WordPress pages"*__

- Verify WordPress REST API is accessible
- Check if the site URL is correct

__"Google API error: QUOTA_EXCEEDED"*__

- Reduce requests per second with --rps 2
- Wait for quota to reset (usually next day)

__Slow performance on large sites__

- Reduce --rps to avoid API timeouts
- Consider analyzing in batches

### License

MIT License

### Contributing

Contributions are welcome! Please submit pull requests or open issues for bugs and feature requests.

### Support

For issues or questions:

- Check the troubleshooting section
- Review Google PageSpeed Insights [documentation](https://developers.google.com/speed/docs/insights/v5/get-started)
- Open an issue on GitHub
