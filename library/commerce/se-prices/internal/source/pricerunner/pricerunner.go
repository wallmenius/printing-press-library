// Package pricerunner extracts structured product, category, search, and deal
// data from PriceRunner.se's server-rendered HTML pages.
//
// PriceRunner embeds its server-side state as raw JSON inside a script tag:
//
//	<script id="initial_payload" type="application/json">{"...":{...}}</script>
//
// Two top-level keys carry the useful data:
//
//	__INITIAL_PROPS__         — per-page framework state (mostly chrome)
//	__DEHYDRATED_QUERY_STATE__ — the React Query cache; this is where the
//	                            real product/search/offer payloads live
//
// Search results live under queryKey ["serp-search", "SE", {"query": "..."}]
// with data.pages[0].products (paginated React Query infinite query).
//
// Product details live across three queries:
//   - ["product-detail-initial", "SE", "<group>", "<id>"] — product, brand, category
//   - ["product-detail-offers",  "SE", "<id>", {...}]     — offers, merchants
//   - ["product-price-level", "<id>", null]               — priceLevel/change
package pricerunner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	// HostPriceRunner is the public web host used for all routes.
	HostPriceRunner = "https://www.pricerunner.se"
)

// initialPayloadRe captures the JSON inside <script id="initial_payload">.
var initialPayloadRe = regexp.MustCompile(`(?s)<script id="initial_payload"[^>]*>(.+?)</script>`)

// ExtractInitialPayload returns the raw JSON bytes from PriceRunner's
// initial_payload script.
func ExtractInitialPayload(html []byte) ([]byte, error) {
	m := initialPayloadRe.FindSubmatch(html)
	if m == nil {
		return nil, fmt.Errorf("pricerunner: no initial_payload script found in HTML")
	}
	return m[1], nil
}

// QueryEntry mirrors one React Query cache entry inside __DEHYDRATED_QUERY_STATE__.
type QueryEntry struct {
	QueryKey json.RawMessage `json:"queryKey"`
	State    struct {
		Data json.RawMessage `json:"data"`
	} `json:"state"`
}

// queryEnvelope is the dehydrated React Query cache shape.
type queryEnvelope struct {
	Queries []QueryEntry `json:"queries"`
}

// LoadQueries returns the React Query entries from the page payload.
func LoadQueries(html []byte) ([]QueryEntry, error) {
	payload, err := ExtractInitialPayload(html)
	if err != nil {
		return nil, err
	}
	var top struct {
		Dehydrated json.RawMessage `json:"__DEHYDRATED_QUERY_STATE__"`
	}
	if err := json.Unmarshal(payload, &top); err != nil {
		return nil, fmt.Errorf("pricerunner: parsing initial_payload top-level: %w", err)
	}
	var env queryEnvelope
	if err := json.Unmarshal(top.Dehydrated, &env); err != nil {
		return nil, fmt.Errorf("pricerunner: parsing __DEHYDRATED_QUERY_STATE__: %w", err)
	}
	return env.Queries, nil
}

// FindQueryByKind returns the first query whose queryKey starts with the given
// kind string (e.g., "serp-search", "product-detail-initial").
func FindQueryByKind(queries []QueryEntry, kind string) (json.RawMessage, error) {
	for _, q := range queries {
		var key []json.RawMessage
		if err := json.Unmarshal(q.QueryKey, &key); err != nil {
			continue
		}
		if len(key) == 0 {
			continue
		}
		var first string
		if err := json.Unmarshal(key[0], &first); err != nil {
			continue
		}
		if first == kind {
			return q.State.Data, nil
		}
	}
	return nil, fmt.Errorf("pricerunner: no query with kind %q", kind)
}

// Product is the typed view of a PriceRunner product page.
type Product struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	URL            string          `json:"url,omitempty"`
	Brand          string          `json:"brand,omitempty"`
	BrandID        string          `json:"brand_id,omitempty"`
	Category       string          `json:"category,omitempty"`
	CategoryID     string          `json:"category_id,omitempty"`
	Manufacturer   string          `json:"manufacturer,omitempty"`
	Description    string          `json:"description,omitempty"`
	ImageURL       string          `json:"image_url,omitempty"`
	LowestPriceSEK float64         `json:"lowest_price_sek,omitempty"`
	OfferCount     int             `json:"offer_count,omitempty"`
	StockStatus    string          `json:"stock_status,omitempty"`
	Rating         float64         `json:"rating,omitempty"`
	ReviewCount    int             `json:"review_count,omitempty"`
	EAN            string          `json:"ean,omitempty"`
	SKU            string          `json:"sku,omitempty"`
	BreadCrumbs    json.RawMessage `json:"breadcrumbs,omitempty"`
	PriceLevel     string          `json:"price_level,omitempty"`
	PriceChange    json.RawMessage `json:"price_change,omitempty"`
	Offers         []Offer         `json:"offers,omitempty"`
	OffersSummary  json.RawMessage `json:"offers_summary,omitempty"`
	Merchants      json.RawMessage `json:"merchants,omitempty"`
	ReviewSummary  json.RawMessage `json:"review_summary,omitempty"`
	Specs          json.RawMessage `json:"specs,omitempty"`
}

// Offer is one merchant's listed price.
type Offer struct {
	MerchantID string  `json:"merchant_id,omitempty"`
	Merchant   string  `json:"merchant,omitempty"`
	PriceSEK   float64 `json:"price_sek"`
	Shipping   float64 `json:"shipping_sek,omitempty"`
	TotalSEK   float64 `json:"total_sek,omitempty"`
	Stock      string  `json:"stock,omitempty"`
	URL        string  `json:"url,omitempty"`
	Currency   string  `json:"currency,omitempty"`
	IsKlarna   bool    `json:"is_klarna,omitempty"`
	Condition  string  `json:"condition,omitempty"`
}

type rawProductDetailInitial struct {
	Product *struct {
		ID            any     `json:"id"`
		Name          string  `json:"name"`
		Description   string  `json:"description"`
		URL           string  `json:"url"`
		Image         string  `json:"image"`
		ImageURL      string  `json:"imageUrl"`
		Manufacturer  string  `json:"manufacturer"`
		EAN           string  `json:"ean"`
		Gtin          string  `json:"gtin"`
		SKU           string  `json:"sku"`
		Rating        float64 `json:"rating"`
		AverageRating float64 `json:"averageRating"`
		ReviewCount   int     `json:"reviewCount"`
		LowestPrice   float64 `json:"lowestPrice"`
	} `json:"product"`
	Brand         *brandObj       `json:"brand"`
	Category      *categoryObj    `json:"category"`
	BreadCrumbs   json.RawMessage `json:"breadCrumbs"`
	ReviewSummary json.RawMessage `json:"reviewSummary"`
}

type brandObj struct {
	ID   any    `json:"id"`
	Name string `json:"name"`
}

type categoryObj struct {
	ID   any    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type rawProductOffersData struct {
	Offers         []rawDetailOffer `json:"offers"`
	StaticOffers   []rawDetailOffer `json:"staticOffers"`
	ExcludedOffers []rawDetailOffer `json:"excludedOffers"`
	OffersSummary  json.RawMessage  `json:"offersSummary"`
	Merchants      json.RawMessage  `json:"merchants"`
}

// money is the {amount: "X.XX", currency: "SEK"} shape PriceRunner uses.
type money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func (m money) Float() float64 {
	if m.Amount == "" {
		return 0
	}
	var f float64
	if _, err := fmt.Sscanf(m.Amount, "%f", &f); err != nil {
		return 0
	}
	return f
}

type rawDetailOffer struct {
	ID           any    `json:"id"`
	MerchantID   any    `json:"merchantId"`
	MerchantName string `json:"merchantName"`
	ShopName     string `json:"shopName"`
	Name         string `json:"name"`
	Price        *money `json:"price"`
	ShippingCost *money `json:"shippingCost"`
	StockStatus  string `json:"stockStatus"`
	Availability string `json:"availability"`
	URL          string `json:"url"`
	Link         string `json:"link"`
	DeepLink     string `json:"deepLink"`
	Currency     string `json:"currency"`
	IsKlarna     bool   `json:"isKlarna"`
	Condition    string `json:"condition"`
}

type rawPriceLevel struct {
	PriceLevel  string          `json:"priceLevel"`
	PriceChange json.RawMessage `json:"priceChange"`
}

// ParseProduct extracts a typed Product from a /pl/<id>/<...> HTML byte blob.
// It walks __DEHYDRATED_QUERY_STATE__ for the product-detail-initial,
// product-detail-offers, and product-price-level queries.
func ParseProduct(html []byte) (*Product, error) {
	queries, err := LoadQueries(html)
	if err != nil {
		return nil, err
	}
	initial, err := FindQueryByKind(queries, "product-detail-initial")
	if err != nil {
		return nil, fmt.Errorf("pricerunner: %w (product page may have changed shape)", err)
	}
	var rpdi rawProductDetailInitial
	if err := json.Unmarshal(initial, &rpdi); err != nil {
		return nil, fmt.Errorf("pricerunner: parsing product-detail-initial: %w", err)
	}
	p := &Product{}
	if rpdi.Product != nil {
		p.ID = anyToString(rpdi.Product.ID)
		p.Name = rpdi.Product.Name
		p.Description = rpdi.Product.Description
		p.URL = rpdi.Product.URL
		p.ImageURL = firstNonEmptyStr(rpdi.Product.ImageURL, rpdi.Product.Image)
		p.Manufacturer = rpdi.Product.Manufacturer
		p.EAN = firstNonEmptyStr(rpdi.Product.EAN, rpdi.Product.Gtin)
		p.SKU = rpdi.Product.SKU
		p.Rating = firstNonZeroFloat(rpdi.Product.AverageRating, rpdi.Product.Rating)
		p.ReviewCount = rpdi.Product.ReviewCount
		p.LowestPriceSEK = rpdi.Product.LowestPrice
	}
	if rpdi.Brand != nil {
		p.Brand = rpdi.Brand.Name
		p.BrandID = anyToString(rpdi.Brand.ID)
	}
	if rpdi.Category != nil {
		p.Category = rpdi.Category.Name
		p.CategoryID = anyToString(rpdi.Category.ID)
	}
	p.BreadCrumbs = rpdi.BreadCrumbs
	p.ReviewSummary = rpdi.ReviewSummary

	if offersData, err := FindQueryByKind(queries, "product-detail-offers"); err == nil {
		var rpod rawProductOffersData
		if jerr := json.Unmarshal(offersData, &rpod); jerr == nil {
			p.OffersSummary = rpod.OffersSummary
			p.Merchants = rpod.Merchants
			merchantNames := parseMerchantsLookup(rpod.Merchants)
			p.Offers = mergeOffers(rpod.Offers, rpod.StaticOffers)
			for i, o := range p.Offers {
				if o.Merchant == "" && o.MerchantID != "" {
					if name, ok := merchantNames[o.MerchantID]; ok {
						p.Offers[i].Merchant = name
					}
				}
			}
			if p.LowestPriceSEK == 0 && len(p.Offers) > 0 {
				p.LowestPriceSEK = p.Offers[0].PriceSEK
			}
			p.OfferCount = len(p.Offers)
		}
	}

	if levelData, err := FindQueryByKind(queries, "product-price-level"); err == nil {
		var rpl rawPriceLevel
		if jerr := json.Unmarshal(levelData, &rpl); jerr == nil {
			p.PriceLevel = rpl.PriceLevel
			p.PriceChange = rpl.PriceChange
		}
	}
	return p, nil
}

func mergeOffers(primary, static []rawDetailOffer) []Offer {
	all := append([]rawDetailOffer{}, primary...)
	all = append(all, static...)
	out := make([]Offer, 0, len(all))
	for _, o := range all {
		off := Offer{
			MerchantID: anyToString(o.MerchantID),
			Merchant:   firstNonEmptyStr(o.MerchantName, o.ShopName),
			Stock:      firstNonEmptyStr(o.StockStatus, o.Availability),
			URL:        firstNonEmptyStr(o.URL, o.DeepLink, o.Link),
			Currency:   firstNonEmptyStr(o.Currency, "SEK"),
			IsKlarna:   o.IsKlarna,
			Condition:  o.Condition,
		}
		if off.URL != "" && !strings.HasPrefix(off.URL, "http") {
			off.URL = HostPriceRunner + off.URL
		}
		if o.Price != nil {
			off.PriceSEK = o.Price.Float()
			if off.Currency == "" {
				off.Currency = o.Price.Currency
			}
		}
		if o.ShippingCost != nil {
			off.Shipping = o.ShippingCost.Float()
		}
		off.TotalSEK = off.PriceSEK + off.Shipping
		out = append(out, off)
	}
	return out
}

// SearchListing is the parsed result of /results?q=<q>.
type SearchListing struct {
	Query        string          `json:"query"`
	Total        int             `json:"total,omitempty"`
	NumberOfHits int             `json:"number_of_hits,omitempty"`
	Categories   json.RawMessage `json:"categories,omitempty"`
	Spelling     string          `json:"spelling_suggestion,omitempty"`
	Products     []SearchProduct `json:"products"`
}

// SearchProduct is one entry in a search/category result list.
type SearchProduct struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	URL            string  `json:"url,omitempty"`
	Description    string  `json:"description,omitempty"`
	ImageURL       string  `json:"image_url,omitempty"`
	Brand          string  `json:"brand,omitempty"`
	Category       string  `json:"category,omitempty"`
	LowestPriceSEK float64 `json:"lowest_price_sek,omitempty"`
	OfferCount     int     `json:"offer_count,omitempty"`
	Rating         float64 `json:"rating,omitempty"`
	StockStatus    string  `json:"stock_status,omitempty"`
	PriceDrop      float64 `json:"price_drop_pct,omitempty"`
	Rank           int     `json:"rank,omitempty"`
}

type rawSearchPage struct {
	SearchQuery        string             `json:"searchQuery"`
	NumberOfHits       json.RawMessage    `json:"numberOfHits"`
	Products           []rawSearchProduct `json:"products"`
	Categories         json.RawMessage    `json:"categories"`
	SpellingSuggestion string             `json:"spellingSuggestion"`
}

func (p rawSearchPage) numHits() int {
	if len(p.NumberOfHits) == 0 {
		return 0
	}
	// May be int or object {value: N, displayValue: "..."}
	var n int
	if err := json.Unmarshal(p.NumberOfHits, &n); err == nil {
		return n
	}
	var obj struct {
		Value int `json:"value"`
	}
	if err := json.Unmarshal(p.NumberOfHits, &obj); err == nil {
		return obj.Value
	}
	return 0
}

type rawSearchProduct struct {
	ID          any    `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Image       *struct {
		Path string `json:"path"`
		URL  string `json:"url"`
	} `json:"image"`
	Brand    json.RawMessage `json:"brand"`
	Category *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"category"`
	LowestPrice *money `json:"lowestPrice"`
	OfferCount  int    `json:"offerCount"`
	Rating      *struct {
		NumberOfRatings int    `json:"numberOfRatings"`
		AverageRating   string `json:"averageRating"`
		Count           int    `json:"count"`
		Average         string `json:"average"`
	} `json:"rating"`
	StockStatus string          `json:"stockStatus"`
	PriceDrop   json.RawMessage `json:"priceDrop"`
	Rank        *struct {
		Rank  int    `json:"rank"`
		Trend string `json:"trend"`
	} `json:"rank"`
}

// ParseSearch extracts a typed SearchListing from a /results page.
func ParseSearch(html []byte, query string) (*SearchListing, error) {
	queries, err := LoadQueries(html)
	if err != nil {
		// Fall back to link extraction so callers always get something
		return &SearchListing{Query: query, Products: extractFallbackLinks(html)}, nil
	}
	serp, err := FindQueryByKind(queries, "serp-search")
	if err != nil {
		return &SearchListing{Query: query, Products: extractFallbackLinks(html)}, nil
	}
	var infq struct {
		Pages []rawSearchPage `json:"pages"`
	}
	if err := json.Unmarshal(serp, &infq); err != nil {
		return &SearchListing{Query: query, Products: extractFallbackLinks(html)}, nil
	}
	out := &SearchListing{Query: query}
	for _, p := range infq.Pages {
		if out.NumberOfHits == 0 {
			out.NumberOfHits = p.numHits()
			out.Total = out.NumberOfHits
			out.Categories = p.Categories
			out.Spelling = p.SpellingSuggestion
		}
		for _, e := range p.Products {
			sp := SearchProduct{
				ID:          anyToString(e.ID),
				Name:        e.Name,
				URL:         e.URL,
				Description: e.Description,
				StockStatus: e.StockStatus,
				OfferCount:  e.OfferCount,
			}
			if sp.URL != "" && !strings.HasPrefix(sp.URL, "http") {
				sp.URL = HostPriceRunner + sp.URL
			}
			if e.Image != nil {
				sp.ImageURL = firstNonEmptyStr(e.Image.URL, e.Image.Path)
			}
			if e.Category != nil {
				sp.Category = e.Category.Name
			}
			// Brand may be string or {id,name} object — try both
			var asStr string
			if json.Unmarshal(e.Brand, &asStr) == nil && asStr != "" {
				sp.Brand = asStr
			} else {
				var asObj struct {
					Name string `json:"name"`
				}
				if json.Unmarshal(e.Brand, &asObj) == nil {
					sp.Brand = asObj.Name
				}
			}
			if e.LowestPrice != nil {
				sp.LowestPriceSEK = e.LowestPrice.Float()
			}
			if e.Rating != nil {
				sp.Rating = parseRatingString(firstNonEmptyStr(e.Rating.AverageRating, e.Rating.Average))
			}
			if e.Rank != nil {
				sp.Rank = e.Rank.Rank
			}
			out.Products = append(out.Products, sp)
		}
	}
	if len(out.Products) == 0 {
		out.Products = extractFallbackLinks(html)
	}
	return out, nil
}

var fallbackLinkRe = regexp.MustCompile(`href="(/pl/[0-9]+-[0-9]+/[^"]+)"[^>]*>([^<]+)<`)
var fallbackIDRe = regexp.MustCompile(`/pl/[0-9]+-([0-9]+)/`)

func extractFallbackLinks(html []byte) []SearchProduct {
	matches := fallbackLinkRe.FindAllSubmatch(html, -1)
	seen := make(map[string]struct{})
	var out []SearchProduct
	for _, m := range matches {
		path := string(m[1])
		if _, dup := seen[path]; dup {
			continue
		}
		seen[path] = struct{}{}
		idMatch := fallbackIDRe.FindStringSubmatch(path)
		id := ""
		if len(idMatch) >= 2 {
			id = idMatch[1]
		}
		name := strings.TrimSpace(string(m[2]))
		if name == "" {
			continue
		}
		out = append(out, SearchProduct{
			ID:   id,
			Name: name,
			URL:  HostPriceRunner + path,
		})
		if len(out) >= 50 {
			break
		}
	}
	return out
}

// CategoryListing is the parsed result of /cl/<id>/<slug>.
type CategoryListing struct {
	ID       int             `json:"id,omitempty"`
	Slug     string          `json:"slug,omitempty"`
	Name     string          `json:"name,omitempty"`
	Total    int             `json:"total,omitempty"`
	Products []SearchProduct `json:"products"`
}

// ParseCategory extracts a typed CategoryListing. Category pages reuse the
// serp-search-shaped query under a different key when paginated. Falls back to
// the link-extraction path so callers always get something usable.
func ParseCategory(html []byte) (*CategoryListing, error) {
	sl, err := ParseSearch(html, "")
	if err != nil {
		return &CategoryListing{Products: extractFallbackLinks(html)}, nil
	}
	out := &CategoryListing{Total: sl.Total}
	out.Products = sl.Products
	if len(out.Products) == 0 {
		out.Products = extractFallbackLinks(html)
	}
	return out, nil
}

// parseMerchantsLookup turns the dict-of-merchants payload into id -> name.
// PriceRunner returns merchants as a JSON object keyed by merchant ID.
func parseMerchantsLookup(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v.Name
	}
	return out
}

func parseRatingString(s string) float64 {
	if s == "" {
		return 0
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return 0
	}
	return f
}

// Deal is one curated /deals entry.
type Deal struct {
	ID          string  `json:"id,omitempty"`
	Title       string  `json:"title"`
	URL         string  `json:"url,omitempty"`
	ImageURL    string  `json:"image_url,omitempty"`
	Merchant    string  `json:"merchant,omitempty"`
	PriceSEK    float64 `json:"price_sek,omitempty"`
	OldPriceSEK float64 `json:"old_price_sek,omitempty"`
	DiscountPct int     `json:"discount_pct,omitempty"`
	Category    string  `json:"category,omitempty"`
}

// ParseDeals extracts deals from the /deals page. PriceRunner's deals shape is
// volatile across releases; this parser walks known deal-shaped query keys and
// tolerates missing data with an empty result. Falls back to the bare /pl/
// product-link list if no structured deals data is available.
func ParseDeals(html []byte) ([]Deal, error) {
	queries, err := LoadQueries(html)
	if err != nil {
		return []Deal{}, nil
	}
	for _, kind := range []string{"deals-page", "deals-list", "campaigns-list", "promotions-list"} {
		data, err := FindQueryByKind(queries, kind)
		if err != nil {
			continue
		}
		var dpl struct {
			Deals []rawDeal `json:"deals"`
			Items []rawDeal `json:"items"`
		}
		if jerr := json.Unmarshal(data, &dpl); jerr != nil {
			continue
		}
		entries := dpl.Deals
		if len(entries) == 0 {
			entries = dpl.Items
		}
		out := make([]Deal, 0, len(entries))
		for _, d := range entries {
			out = append(out, Deal{
				ID:          anyToString(d.ID),
				Title:       firstNonEmptyStr(d.Title, d.Name),
				URL:         firstNonEmptyStr(d.URL, d.Link),
				ImageURL:    firstNonEmptyStr(d.ImageURL, d.Image),
				Merchant:    firstNonEmptyStr(d.Merchant, d.ShopName),
				PriceSEK:    firstNonZeroFloat(d.PriceSEK, d.Price),
				OldPriceSEK: firstNonZeroFloat(d.OldPriceSEK, d.OldPrice),
				DiscountPct: d.DiscountPct,
				Category:    firstNonEmptyStr(d.CategoryName, d.Category),
			})
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	return []Deal{}, nil
}

type rawDeal struct {
	ID           any     `json:"id"`
	Title        string  `json:"title"`
	Name         string  `json:"name"`
	URL          string  `json:"url"`
	Link         string  `json:"link"`
	ImageURL     string  `json:"imageUrl"`
	Image        string  `json:"image"`
	Merchant     string  `json:"merchant"`
	ShopName     string  `json:"shopName"`
	PriceSEK     float64 `json:"priceSek"`
	Price        float64 `json:"price"`
	OldPriceSEK  float64 `json:"oldPriceSek"`
	OldPrice     float64 `json:"oldPrice"`
	DiscountPct  int     `json:"discountPct"`
	CategoryName string  `json:"categoryName"`
	Category     string  `json:"category"`
}

func anyToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	case json.Number:
		return x.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstNonZeroFloat(values ...float64) float64 {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}
