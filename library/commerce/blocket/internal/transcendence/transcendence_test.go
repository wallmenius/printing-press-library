package transcendence

import "testing"

func TestMedianInt(t *testing.T) {
	cases := []struct {
		name string
		in   []int
		want int
	}{
		{"empty", nil, 0},
		{"single", []int{5}, 5},
		{"odd count sorted", []int{1, 2, 3}, 2},
		{"odd count unsorted", []int{3, 1, 2}, 2},
		{"even count", []int{1, 2, 3, 4}, 2}, // (2+3)/2 = 2 with integer truncation
		{"five-element", []int{10, 20, 30, 40, 50}, 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MedianInt(tc.in)
			if got != tc.want {
				t.Fatalf("MedianInt(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestPercentileInt(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		p10, p50, p90 := PercentileInt(nil)
		if p10 != 0 || p50 != 0 || p90 != 0 {
			t.Fatalf("expected all zeros, got %d/%d/%d", p10, p50, p90)
		}
	})

	t.Run("ten elements", func(t *testing.T) {
		// 10 elements => idx for p=0.1 is int(0.1*9)=0, p=0.5 is int(4.5)=4, p=0.9 is int(8.1)=8
		vals := []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
		p10, p50, p90 := PercentileInt(vals)
		if p10 != 10 {
			t.Errorf("p10 = %d, want 10", p10)
		}
		if p50 != 50 {
			t.Errorf("p50 = %d, want 50", p50)
		}
		if p90 != 90 {
			t.Errorf("p90 = %d, want 90", p90)
		}
	})

	t.Run("unsorted input is sorted internally", func(t *testing.T) {
		vals := []int{100, 50, 10, 90, 30, 70, 20, 80, 40, 60}
		p10, p50, p90 := PercentileInt(vals)
		if p10 != 10 || p50 != 50 || p90 != 90 {
			t.Fatalf("got %d/%d/%d, want 10/50/90", p10, p50, p90)
		}
	})
}

func TestMileageBand(t *testing.T) {
	cases := map[int]string{
		0:      "unknown",
		-1:     "unknown",
		1000:   "0-5k",
		4999:   "0-5k",
		5000:   "5k-10k",
		9999:   "5k-10k",
		10000:  "10k-15k",
		14999:  "10k-15k",
		15000:  "15k-20k",
		19999:  "15k-20k",
		20000:  "20k-30k",
		29999:  "20k-30k",
		30000:  "30k+",
		100000: "30k+",
	}
	for in, want := range cases {
		got := MileageBand(in)
		if got != want {
			t.Errorf("MileageBand(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestYearBand(t *testing.T) {
	cases := map[int]string{
		0:    "unknown",
		-1:   "unknown",
		2019: "2019-2021", // (2019/3)*3 = 2019
		2020: "2019-2021", // (2020/3)*3 = 2019
		2021: "2019-2021", // (2021/3)*3 = 2019, since 2021/3 = 673
		2022: "2022-2024", // (2022/3)*3 = 2022
		2023: "2022-2024",
		2024: "2022-2024",
		2025: "2025-2027",
	}
	for in, want := range cases {
		got := YearBand(in)
		if got != want {
			t.Errorf("YearBand(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestVerticalTable(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"car":         "cars",
		"cars":        "cars",
		"CARS":        "cars",
		"  cars  ":    "cars",
		"ad":          "ads",
		"ads":         "ads",
		"bap":         "ads",
		"motorcycles": "resources",
		"boats":       "resources",
		"caravan":     "resources",
	}
	for in, want := range cases {
		got := VerticalTable(in)
		if got != want {
			t.Errorf("VerticalTable(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnmarshalAdRow(t *testing.T) {
	t.Run("full payload", func(t *testing.T) {
		raw := []byte(`{
			"heading": "Volvo V70 D5",
			"make": "Volvo",
			"model": "V70",
			"year": 2015,
			"mileage": 120000,
			"price": {"amount": 89000, "currency_code": "SEK"},
			"location": "Stockholm",
			"coordinates": {"lat": 59.33, "lon": 18.06},
			"timestamp": 1700000000,
			"org_id": 4242,
			"organisation_name": "Bil AB",
			"canonical_url": "https://www.blocket.se/ad/abc"
		}`)
		row, err := UnmarshalAdRow("abc-123", raw, "cars")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if row.AdID != "abc-123" {
			t.Errorf("AdID = %q, want abc-123", row.AdID)
		}
		if row.Make != "Volvo" || row.Model != "V70" || row.Year != 2015 || row.Mileage != 120000 {
			t.Errorf("vehicle fields wrong: %+v", row)
		}
		if row.PriceAmount != 89000 || row.PriceCurr != "SEK" {
			t.Errorf("price fields wrong: %+v", row)
		}
		if row.Lat != 59.33 || row.Lon != 18.06 {
			t.Errorf("coords wrong: %+v", row)
		}
		if row.OrgID != 4242 || row.OrgName != "Bil AB" {
			t.Errorf("org fields wrong: %+v", row)
		}
		if row.Vertical != "cars" {
			t.Errorf("vertical = %q, want cars", row.Vertical)
		}
	})

	t.Run("ad_id from numeric payload when id arg empty", func(t *testing.T) {
		// raw.AdID is json.Number, so only numeric payload AdIDs are accepted.
		raw := []byte(`{"ad_id": 9876, "heading": "X"}`)
		row, err := UnmarshalAdRow("", raw, "ads")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if row.AdID != "9876" {
			t.Errorf("AdID = %q, want 9876", row.AdID)
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		_, err := UnmarshalAdRow("x", []byte("not json"), "ads")
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})
}
