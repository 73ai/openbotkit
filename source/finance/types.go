package finance

type Config struct{}

type Quote struct {
	Symbol                     string  `json:"symbol"`
	ShortName                  string  `json:"shortName"`
	QuoteType                  string  `json:"quoteType"`
	Currency                   string  `json:"currency"`
	Exchange                   string  `json:"exchange"`
	RegularMarketPrice         float64 `json:"regularMarketPrice"`
	RegularMarketChange        float64 `json:"regularMarketChange"`
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
	RegularMarketDayHigh       float64 `json:"regularMarketDayHigh"`
	RegularMarketDayLow        float64 `json:"regularMarketDayLow"`
	RegularMarketVolume        int64   `json:"regularMarketVolume"`
	MarketState                string  `json:"marketState"`
}
