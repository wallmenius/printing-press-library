// Package prisjakt extracts structured product, category, and search data from
// Prisjakt.nu's server-rendered HTML pages.
//
// Prisjakt embeds its full React Query cache in the HTML as a JS assignment:
//
//	<script>window.__REACT_QUERY_STATE__ = JSON.parse('{"queries":[...]}');</script>
//
// We extract the JSON literal (with its JS-string escapes), decode the escapes
// back to UTF-8, and parse the resulting JSON. Each query in the cache has a
// queryKey (route-typed) and a state.data payload. The shape varies per page:
// product pages carry a `product` key; category pages carry a
// `productCollection` key (which carries `trendingProducts` and `slices` as the
// publicly-visible product surface — the full paginated list is a subsequent
// client-side fetch we cannot reach).
package prisjakt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// HostPrisjakt is the public web host used for all routes.
	HostPrisjakt = "https://www.prisjakt.nu"
)

// reactQueryRe captures the body of `window.__REACT_QUERY_STATE__ = JSON.parse('...')`.
var reactQueryRe = regexp.MustCompile(`(?s)window\.__REACT_QUERY_STATE__\s*=\s*JSON\.parse\('(.+?)'\)`)

// FlexID accepts either a JSON string or a JSON number and stores both as
// strings. Used because Prisjakt mixes int IDs (brand, product) and string
// IDs (category, productCollection UUIDs).
type FlexID string

// UnmarshalJSON accepts both quoted strings and bare numbers.
func (f *FlexID) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = FlexID(s)
		return nil
	}
	*f = FlexID(strings.TrimSpace(string(b)))
	return nil
}

// String renders the ID without quotes.
func (f FlexID) String() string { return string(f) }

// Int parses the ID as an integer; returns 0 on parse failure.
func (f FlexID) Int() int {
	n, _ := strconv.Atoi(string(f))
	return n
}

// MarshalJSON renders the ID as a string when written back out — gives stable
// types in --json output, regardless of input shape.
func (f FlexID) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(f))
}

// ExtractReactQueryState pulls the React Query JSON state out of an HTML page.
// Returns the decoded JSON bytes ready for json.Unmarshal.
func ExtractReactQueryState(html []byte) ([]byte, error) {
	m := reactQueryRe.FindSubmatch(html)
	if m == nil {
		return nil, fmt.Errorf("prisjakt: no __REACT_QUERY_STATE__ found in HTML (page may be a Cloudflare challenge or unrelated route)")
	}
	raw := string(m[1])
	unescaped := decodeJSStringLiteral(raw)
	return []byte(unescaped), nil
}

// decodeJSStringLiteral decodes the escape sequences inside a JS string literal
// emitted by JSON.parse('...') wrappers. Handles \', \\, \", \n, \r, \t, \b,
// \f, \xNN, \uNNNN. The output is the literal characters, ready to be
// interpreted as JSON.
func decodeJSStringLiteral(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		next := s[i+1]
		switch next {
		case '\\':
			b.WriteByte('\\')
			i++
		case '\'':
			b.WriteByte('\'')
			i++
		case '"':
			b.WriteByte('"')
			i++
		case 'n':
			b.WriteByte('\n')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case 'b':
			b.WriteByte('\b')
			i++
		case 'f':
			b.WriteByte('\f')
			i++
		case '/':
			b.WriteByte('/')
			i++
		case 'x':
			if i+3 < len(s) {
				if v, err := strconv.ParseUint(s[i+2:i+4], 16, 8); err == nil {
					b.WriteByte(byte(v))
					i += 3
					continue
				}
			}
			b.WriteByte(c)
		case 'u':
			if i+5 < len(s) {
				if v, err := strconv.ParseUint(s[i+2:i+6], 16, 32); err == nil {
					b.WriteRune(rune(v))
					i += 5
					continue
				}
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// QueryEnvelope mirrors the React Query cache shape:
//
//	{ "mutations": [...], "queries": [{ "queryKey": [...], "state": {"data": {...}} }] }
type QueryEnvelope struct {
	Queries []QueryEntry `json:"queries"`
}

// QueryEntry is one cached query with its key and resolved data.
type QueryEntry struct {
	QueryKey json.RawMessage `json:"queryKey"`
	State    struct {
		Data json.RawMessage `json:"data"`
	} `json:"state"`
}

// FindQuery returns the first query whose first key element matches `kind`.
func (e *QueryEnvelope) FindQuery(kind string) (json.RawMessage, error) {
	for _, q := range e.Queries {
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
	return nil, fmt.Errorf("prisjakt: no query with kind %q in React Query state", kind)
}

// ParseQueryEnvelope parses the decoded React Query JSON into an envelope.
func ParseQueryEnvelope(jsonBytes []byte) (*QueryEnvelope, error) {
	var env QueryEnvelope
	if err := json.Unmarshal(jsonBytes, &env); err != nil {
		return nil, fmt.Errorf("prisjakt: parsing React Query state: %w", err)
	}
	return &env, nil
}

// Product is the typed view of Prisjakt's product object.
type Product struct {
	ID                int             `json:"id"`
	Name              string          `json:"name"`
	Description       string          `json:"description,omitempty"`
	PathName          string          `json:"path_name,omitempty"`
	StockStatus       string          `json:"stock_status,omitempty"`
	ReleaseDate       string          `json:"release_date,omitempty"`
	IndexableIn       []string        `json:"indexable_in,omitempty"`
	URL               string          `json:"url,omitempty"`
	Brand             *Brand          `json:"brand,omitempty"`
	Category          *Category       `json:"category,omitempty"`
	PriceSummary      *PriceSummary   `json:"price_summary,omitempty"`
	Offers            []Offer         `json:"offers,omitempty"`
	MobileContracts   json.RawMessage `json:"mobile_contracts,omitempty"`
	Sparkline         json.RawMessage `json:"sparkline,omitempty"`
	UserReviewSummary json.RawMessage `json:"user_review_summary,omitempty"`
	AggregatedRating  json.RawMessage `json:"aggregated_rating,omitempty"`
	CoreProperties    json.RawMessage `json:"core_properties,omitempty"`
	Media             json.RawMessage `json:"media,omitempty"`
	DealInfo          json.RawMessage `json:"deal_info,omitempty"`
	ExpertContent     json.RawMessage `json:"expert_content,omitempty"`
	Variants          json.RawMessage `json:"variants,omitempty"`
	Relations         json.RawMessage `json:"relations,omitempty"`
	Popular           json.RawMessage `json:"popular_products,omitempty"`
	Trending          json.RawMessage `json:"trending_products,omitempty"`
	OthersVisited     json.RawMessage `json:"others_visited,omitempty"`
	SanityFaq         json.RawMessage `json:"sanity_faq,omitempty"`
	Campaigns         json.RawMessage `json:"campaigns,omitempty"`
	VerifiedBadge     json.RawMessage `json:"verified_badge,omitempty"`
	SanityBadges      json.RawMessage `json:"sanity_badges,omitempty"`
	IsExpertTopRated  bool            `json:"is_expert_top_rated,omitempty"`
	Popularity        json.RawMessage `json:"popularity,omitempty"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	EAN               string          `json:"ean,omitempty"`
}

// Brand is a Prisjakt brand object.
type Brand struct {
	ID       FlexID `json:"id"`
	Name     string `json:"name"`
	Featured bool   `json:"featured,omitempty"`
	Logo     string `json:"logo,omitempty"`
	PathName string `json:"path_name,omitempty"`
}

// Category is a Prisjakt category object. ID is FlexID because Prisjakt mixes
// numeric and string IDs across category levels.
type Category struct {
	ID                FlexID          `json:"id"`
	Name              string          `json:"name"`
	Logo              string          `json:"logo,omitempty"`
	PathName          string          `json:"path_name,omitempty"`
	Path              json.RawMessage `json:"path,omitempty"`
	HasAdultContent   bool            `json:"has_adult_content,omitempty"`
	ProductCollection json.RawMessage `json:"product_collection,omitempty"`
	Products          json.RawMessage `json:"products,omitempty"`
}

// PriceSummary captures lowest/highest price snapshot at fetch time.
type PriceSummary struct {
	Lowest         int    `json:"lowest_sek,omitempty"`
	Highest        int    `json:"highest_sek,omitempty"`
	Currency       string `json:"currency,omitempty"`
	Count          int    `json:"offer_count,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	HasInStock     bool   `json:"has_in_stock,omitempty"`
	InStockLowest  int    `json:"in_stock_lowest_sek,omitempty"`
	OutStockLowest int    `json:"out_of_stock_lowest_sek,omitempty"`
}

// Offer is one merchant's listed price for the product.
type Offer struct {
	ID         string  `json:"id,omitempty"`
	MerchantID int     `json:"merchant_id,omitempty"`
	Merchant   string  `json:"merchant,omitempty"`
	PriceSEK   float64 `json:"price_sek"`
	Shipping   float64 `json:"shipping_sek,omitempty"`
	TotalSEK   float64 `json:"total_sek,omitempty"`
	Stock      string  `json:"stock,omitempty"`
	URL        string  `json:"url,omitempty"`
	Currency   string  `json:"currency,omitempty"`
	UpdatedAt  string  `json:"updated_at,omitempty"`
}

// rawProductPayload mirrors the Prisjakt product object as it appears in
// product.page query data.
type rawProductPayload struct {
	Product json.RawMessage `json:"product"`
}

// ParseProduct extracts a typed Product from a fetched product-page HTML byte
// blob.
func ParseProduct(html []byte) (*Product, error) {
	state, err := ExtractReactQueryState(html)
	if err != nil {
		return nil, err
	}
	env, err := ParseQueryEnvelope(state)
	if err != nil {
		return nil, err
	}
	pageData, err := env.FindQuery("product")
	if err != nil {
		return nil, err
	}
	var rp rawProductPayload
	if err := json.Unmarshal(pageData, &rp); err != nil {
		return nil, fmt.Errorf("prisjakt: parsing product wrapper: %w", err)
	}
	p, err := decodeProduct(rp.Product)
	if err != nil {
		return nil, err
	}
	if p.PathName != "" && p.URL == "" {
		p.URL = HostPrisjakt + p.PathName
	}
	return p, nil
}

// rawProduct is the raw JSON shape Prisjakt emits inside React Query state.
type rawProduct struct {
	ID                 int             `json:"id"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	PathName           string          `json:"pathName"`
	StockStatus        string          `json:"stockStatus"`
	ReleaseDate        string          `json:"releaseDate"`
	IndexableIn        []string        `json:"indexableIn"`
	IsExpertTopRated   bool            `json:"isExpertTopRated"`
	Brand              *Brand          `json:"brand"`
	Category           *Category       `json:"category"`
	UserReviewSummary  json.RawMessage `json:"userReviewSummary"`
	AggregatedRating   json.RawMessage `json:"aggregatedRatingSummary"`
	CoreProperties     json.RawMessage `json:"coreProperties"`
	Media              json.RawMessage `json:"media"`
	DealInfo           json.RawMessage `json:"dealInfo"`
	ExpertContent      json.RawMessage `json:"expertContent"`
	Variants           json.RawMessage `json:"variants"`
	Relations          json.RawMessage `json:"relations"`
	PopularProducts    json.RawMessage `json:"popularProducts"`
	TrendingProducts   json.RawMessage `json:"trendingProducts"`
	OthersVisited      json.RawMessage `json:"othersVisitedProducts"`
	SanityFaq          json.RawMessage `json:"sanityFaq"`
	Campaigns          json.RawMessage `json:"campaigns"`
	VerifiedBadge      json.RawMessage `json:"verifiedProductBadge"`
	SanityBadges       json.RawMessage `json:"sanityBadges"`
	Sparkline          json.RawMessage `json:"sparkline"`
	Popularity         json.RawMessage `json:"popularity"`
	Metadata           json.RawMessage `json:"metadata"`
	PartnerVideos      json.RawMessage `json:"partnerVideos"`
	ProductDescription json.RawMessage `json:"productDescription"`
	InitialStatistics  json.RawMessage `json:"initialStatistics"`
	PriceSummary       *struct {
		Min *struct {
			SEK float64 `json:"sek"`
		} `json:"min"`
		Max *struct {
			SEK float64 `json:"sek"`
		} `json:"max"`
		Currency string `json:"currency"`
		Count    int    `json:"count"`
	} `json:"priceSummary"`
	Prices *struct {
		Meta              json.RawMessage `json:"meta"`
		Nodes             []rawPriceNode  `json:"nodes"`
		MobileContractsV2 json.RawMessage `json:"mobileContractsV2"`
	} `json:"prices"`
}

type rawPriceNode struct {
	Typename      string `json:"__typename"`
	ShopOfferID   string `json:"shopOfferId"`
	Name          string `json:"name"`
	ExternalURI   string `json:"externalUri"`
	PrimaryMarket bool   `json:"primaryMarket"`
	Condition     string `json:"condition"`
	Stock         *struct {
		Status     string `json:"status"`
		StatusText string `json:"statusText"`
	} `json:"stock"`
	Price *struct {
		InclShipping     *float64 `json:"inclShipping"`
		ExclShipping     float64  `json:"exclShipping"`
		OriginalCurrency string   `json:"originalCurrency"`
	} `json:"price"`
	OfferPrices *struct {
		Price *struct {
			ExclShipping float64  `json:"exclShipping"`
			InclShipping *float64 `json:"inclShipping"`
		} `json:"price"`
	} `json:"offerPrices"`
	Store *struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		PathName string `json:"pathName"`
		Currency string `json:"currency"`
	} `json:"store"`
}

func decodeProduct(raw json.RawMessage) (*Product, error) {
	var rp rawProduct
	if err := json.Unmarshal(raw, &rp); err != nil {
		return nil, fmt.Errorf("prisjakt: parsing product: %w", err)
	}
	p := &Product{
		ID:                rp.ID,
		Name:              rp.Name,
		Description:       rp.Description,
		PathName:          rp.PathName,
		StockStatus:       rp.StockStatus,
		ReleaseDate:       rp.ReleaseDate,
		IndexableIn:       rp.IndexableIn,
		IsExpertTopRated:  rp.IsExpertTopRated,
		Brand:             rp.Brand,
		Category:          rp.Category,
		UserReviewSummary: rp.UserReviewSummary,
		AggregatedRating:  rp.AggregatedRating,
		CoreProperties:    rp.CoreProperties,
		Media:             rp.Media,
		DealInfo:          rp.DealInfo,
		ExpertContent:     rp.ExpertContent,
		Variants:          rp.Variants,
		Relations:         rp.Relations,
		Popular:           rp.PopularProducts,
		Trending:          rp.TrendingProducts,
		OthersVisited:     rp.OthersVisited,
		SanityFaq:         rp.SanityFaq,
		Campaigns:         rp.Campaigns,
		VerifiedBadge:     rp.VerifiedBadge,
		SanityBadges:      rp.SanityBadges,
		Sparkline:         rp.Sparkline,
		Popularity:        rp.Popularity,
		Metadata:          rp.Metadata,
	}
	if rp.PriceSummary != nil {
		ps := &PriceSummary{Currency: rp.PriceSummary.Currency, Count: rp.PriceSummary.Count}
		if rp.PriceSummary.Min != nil {
			ps.Lowest = int(rp.PriceSummary.Min.SEK)
		}
		if rp.PriceSummary.Max != nil {
			ps.Highest = int(rp.PriceSummary.Max.SEK)
		}
		p.PriceSummary = ps
	}
	if rp.Prices != nil {
		p.MobileContracts = rp.Prices.MobileContractsV2
		for _, n := range rp.Prices.Nodes {
			o := Offer{ID: n.ShopOfferID, URL: n.ExternalURI, Currency: "SEK"}
			if n.Stock != nil {
				o.Stock = firstNonEmpty(n.Stock.Status, n.Stock.StatusText)
			}
			if n.Store != nil {
				o.MerchantID = n.Store.ID
				o.Merchant = n.Store.Name
				if o.URL == "" {
					o.URL = HostPrisjakt + n.Store.PathName
				}
				if n.Store.Currency != "" {
					o.Currency = n.Store.Currency
				}
			}
			if n.Price != nil {
				o.PriceSEK = n.Price.ExclShipping
				if n.Price.InclShipping != nil {
					o.TotalSEK = *n.Price.InclShipping
					o.Shipping = o.TotalSEK - o.PriceSEK
				}
			}
			if o.TotalSEK == 0 {
				o.TotalSEK = o.PriceSEK + o.Shipping
			}
			p.Offers = append(p.Offers, o)
		}
	}
	return p, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// CategoryListing is the parsed result from a /c/<slug> page.
type CategoryListing struct {
	Slug     string            `json:"slug"`
	Name     string            `json:"name,omitempty"`
	Total    int               `json:"total,omitempty"`
	Source   string            `json:"source,omitempty"`
	Products []CategoryProduct `json:"products"`
}

// CategoryProduct is one entry in a category's product list.
type CategoryProduct struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	URL            string  `json:"url,omitempty"`
	BrandID        FlexID  `json:"brand_id,omitempty"`
	Brand          string  `json:"brand,omitempty"`
	LowestPriceSEK float64 `json:"lowest_price_sek,omitempty"`
	StockStatus    string  `json:"stock_status,omitempty"`
	OfferCount     int     `json:"offer_count,omitempty"`
	ImageURL       string  `json:"image_url,omitempty"`
	Rating         float64 `json:"rating,omitempty"`
}

// rawCategoryPage is the data shape for productCollection.page queries.
// productCollection lacks a paginated product list in SSR; we extract from
// trendingProducts as the publicly-visible product surface.
type rawCategoryPage struct {
	ProductCollection *struct {
		ID               FlexID             `json:"id"`
		Name             string             `json:"name"`
		Slug             string             `json:"slug"`
		URL              string             `json:"url"`
		TrendingProducts []rawCategoryEntry `json:"trendingProducts"`
		Products         []rawCategoryEntry `json:"products"`
	} `json:"productCollection"`
}

type rawCategoryEntry struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Category string `json:"category"`
	Brand    *struct {
		ID   FlexID `json:"id"`
		Name string `json:"name"`
	} `json:"brand"`
	Price        json.RawMessage `json:"price"`
	PriceSummary *struct {
		Min *struct {
			SEK float64 `json:"sek"`
		} `json:"min"`
		OfferCount int `json:"count"`
	} `json:"priceSummary"`
	StockStatus       string `json:"stockStatus"`
	ImageURL          string `json:"imageUrl"`
	UserReviewSummary *struct {
		Rating float64 `json:"rating"`
		Count  int     `json:"count"`
	} `json:"userReviewSummary"`
	AggregatedRating *struct {
		Score float64 `json:"score"`
		Count int     `json:"count"`
	} `json:"aggregatedRating"`
	IsExpertTopRated bool `json:"isExpertTopRated"`
}

// ParseCategory extracts a typed CategoryListing from an HTML byte blob.
// Returns trendingProducts as the listed set when no paginated products are
// present in SSR (the common case on Prisjakt).
func ParseCategory(html []byte) (*CategoryListing, error) {
	state, err := ExtractReactQueryState(html)
	if err != nil {
		return nil, err
	}
	env, err := ParseQueryEnvelope(state)
	if err != nil {
		return nil, err
	}
	pageData, err := env.FindQuery("productCollection")
	if err != nil {
		return nil, err
	}
	var rp rawCategoryPage
	if err := json.Unmarshal(pageData, &rp); err != nil {
		return nil, fmt.Errorf("prisjakt: parsing category wrapper: %w", err)
	}
	if rp.ProductCollection == nil {
		return &CategoryListing{}, nil
	}
	out := &CategoryListing{
		Slug:   rp.ProductCollection.Slug,
		Name:   rp.ProductCollection.Name,
		Source: "trending",
	}
	entries := rp.ProductCollection.Products
	if len(entries) == 0 {
		entries = rp.ProductCollection.TrendingProducts
	} else {
		out.Source = "products"
	}
	out.Total = len(entries)
	for _, e := range entries {
		cp := CategoryProduct{
			ID:          e.ID,
			Name:        e.Name,
			URL:         e.URL,
			StockStatus: e.StockStatus,
			ImageURL:    e.ImageURL,
		}
		if e.Brand != nil {
			cp.BrandID = e.Brand.ID
			cp.Brand = e.Brand.Name
		}
		if e.PriceSummary != nil {
			if e.PriceSummary.Min != nil {
				cp.LowestPriceSEK = e.PriceSummary.Min.SEK
			}
			cp.OfferCount = e.PriceSummary.OfferCount
		}
		if cp.LowestPriceSEK == 0 && len(e.Price) > 0 {
			// Price may be a bare number or an object {sek: N}; try both.
			var asNum float64
			if err := json.Unmarshal(e.Price, &asNum); err == nil {
				cp.LowestPriceSEK = asNum
			} else {
				var asObj struct {
					SEK float64 `json:"sek"`
				}
				if err := json.Unmarshal(e.Price, &asObj); err == nil {
					cp.LowestPriceSEK = asObj.SEK
				}
			}
		}
		if e.AggregatedRating != nil && cp.Rating == 0 {
			cp.Rating = e.AggregatedRating.Score
		}
		if e.UserReviewSummary != nil && cp.Rating == 0 {
			cp.Rating = e.UserReviewSummary.Rating
		}
		if cp.URL != "" && !strings.HasPrefix(cp.URL, "http") {
			cp.URL = HostPrisjakt + cp.URL
		}
		out.Products = append(out.Products, cp)
	}
	return out, nil
}

// SearchListing is the parsed result of /search?search=<q>.
type SearchListing struct {
	Query    string            `json:"query"`
	Total    int               `json:"total,omitempty"`
	Products []CategoryProduct `json:"products"`
}

// ParseSearch extracts a typed SearchListing. Prisjakt's /search page rehydrates
// productCollection-shaped data because search is implemented as a category-style
// view, so the same parser handles both. NOTE: requires Surf transport because
// /search is Cloudflare-challenged for stdlib UAs.
func ParseSearch(html []byte, query string) (*SearchListing, error) {
	cat, err := ParseCategory(html)
	if err != nil {
		return &SearchListing{Query: query}, nil
	}
	return &SearchListing{Query: query, Total: cat.Total, Products: cat.Products}, nil
}
