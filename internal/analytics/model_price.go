package analytics

// ModelPrice is one entry in the static pricing table.
// Costs are in USD per million tokens.
type ModelPrice struct {
	Prefix              string
	InputPerMToken      float64
	OutputPerMToken     float64
	CacheReadPerMToken  float64
	CacheWritePerMToken float64
}

// Cost is the estimated USD cost for a set of token counts.
type Cost struct {
	Usd             float64
	Estimated       bool // true if model had no matching price entry
	UnpricedTokens  int  // tokens for which no price was found
}

// PRICES is the static pricing table (17 entries, TRD §4).
// Matched with strings.HasPrefix(model, Prefix).
var PRICES = []ModelPrice{
	// Anthropic Claude
	{Prefix: "claude-opus-4", InputPerMToken: 15.00, OutputPerMToken: 75.00, CacheReadPerMToken: 1.50, CacheWritePerMToken: 18.75},
	{Prefix: "claude-sonnet-4", InputPerMToken: 3.00, OutputPerMToken: 15.00, CacheReadPerMToken: 0.30, CacheWritePerMToken: 3.75},
	{Prefix: "claude-haiku-4", InputPerMToken: 0.80, OutputPerMToken: 4.00, CacheReadPerMToken: 0.08, CacheWritePerMToken: 1.00},
	{Prefix: "claude-3-5-sonnet", InputPerMToken: 3.00, OutputPerMToken: 15.00, CacheReadPerMToken: 0.30, CacheWritePerMToken: 3.75},
	{Prefix: "claude-3-5-haiku", InputPerMToken: 0.80, OutputPerMToken: 4.00, CacheReadPerMToken: 0.08, CacheWritePerMToken: 1.00},
	{Prefix: "claude-3-opus", InputPerMToken: 15.00, OutputPerMToken: 75.00, CacheReadPerMToken: 1.50, CacheWritePerMToken: 18.75},
	{Prefix: "claude-3-sonnet", InputPerMToken: 3.00, OutputPerMToken: 15.00, CacheReadPerMToken: 0.30, CacheWritePerMToken: 3.75},
	{Prefix: "claude-3-haiku", InputPerMToken: 0.25, OutputPerMToken: 1.25, CacheReadPerMToken: 0.03, CacheWritePerMToken: 0.30},
	// OpenAI GPT
	{Prefix: "gpt-4o", InputPerMToken: 2.50, OutputPerMToken: 10.00, CacheReadPerMToken: 1.25, CacheWritePerMToken: 0},
	{Prefix: "gpt-4-turbo", InputPerMToken: 10.00, OutputPerMToken: 30.00, CacheReadPerMToken: 0, CacheWritePerMToken: 0},
	{Prefix: "gpt-4", InputPerMToken: 30.00, OutputPerMToken: 60.00, CacheReadPerMToken: 0, CacheWritePerMToken: 0},
	{Prefix: "gpt-3.5", InputPerMToken: 0.50, OutputPerMToken: 1.50, CacheReadPerMToken: 0, CacheWritePerMToken: 0},
	{Prefix: "o1", InputPerMToken: 15.00, OutputPerMToken: 60.00, CacheReadPerMToken: 7.50, CacheWritePerMToken: 0},
	{Prefix: "o3", InputPerMToken: 10.00, OutputPerMToken: 40.00, CacheReadPerMToken: 2.50, CacheWritePerMToken: 0},
	// Google Gemini
	{Prefix: "gemini-2.0-flash", InputPerMToken: 0.10, OutputPerMToken: 0.40, CacheReadPerMToken: 0.025, CacheWritePerMToken: 0},
	{Prefix: "gemini-1.5-pro", InputPerMToken: 1.25, OutputPerMToken: 5.00, CacheReadPerMToken: 0.3125, CacheWritePerMToken: 0},
	{Prefix: "gemini-1.5-flash", InputPerMToken: 0.075, OutputPerMToken: 0.30, CacheReadPerMToken: 0.01875, CacheWritePerMToken: 0},
}
