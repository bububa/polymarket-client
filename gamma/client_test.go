package gamma

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGammaIntegerIDEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/markets/123":
			_, _ = w.Write([]byte(`{"id":123,"conditionId":"0xcond","question":"Question?"}`))
		case "/events/234":
			_, _ = w.Write([]byte(`{"id":"234","ticker":"evt"}`))
		case "/series/345":
			_, _ = w.Write([]byte(`{"id":345,"ticker":"series"}`))
		case "/tags/456":
			_, _ = w.Write([]byte(`{"id":"456","label":"tag"}`))
		case "/tags/456/related-tags":
			if r.URL.Query().Get("status") != "active" {
				t.Fatalf("related-tag query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[{"id":1,"tagID":"456","relatedTagID":789,"rank":"2"}]`))
		case "/tags/456/related-tags/tags":
			_, _ = w.Write([]byte(`[{"id":789,"label":"related"}]`))
		case "/events/234/tags":
			_, _ = w.Write([]byte(`[{"id":"456","label":"event-tag","forceHide":true,"isCarousel":true}]`))
		case "/comments/567":
			_, _ = w.Write([]byte(`[{"id":"567","eventId":234}]`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	client := New(Config{Host: srv.URL})
	market := Market{ID: 123}
	if err := client.GetMarket(context.Background(), &market); err != nil || int(market.ID) != 123 {
		t.Fatalf("market=%+v err=%v", market, err)
	}
	event := Event{ID: 234}
	if err := client.GetEvent(context.Background(), &event); err != nil || int(event.ID) != 234 {
		t.Fatalf("event=%+v err=%v", event, err)
	}
	series := Series{ID: 345}
	if err := client.GetSeries(context.Background(), &series); err != nil || int(series.ID) != 345 {
		t.Fatalf("series=%+v err=%v", series, err)
	}
	tag := Tag{ID: 456}
	if err := client.GetTag(context.Background(), &tag); err != nil || int(tag.ID) != 456 {
		t.Fatalf("tag=%+v err=%v", tag, err)
	}
	relationships, err := client.GetRelatedTagRelationships(context.Background(), 456, RelatedTagParams{Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if len(relationships) != 1 || int(relationships[0].TagID) != 456 || int(relationships[0].Rank) != 2 {
		t.Fatalf("relationships=%+v", relationships)
	}
	related, err := client.GetRelatedTags(context.Background(), 456, RelatedTagParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(related) != 1 || int(related[0].ID) != 789 {
		t.Fatalf("related=%+v", related)
	}
	eventTags, err := client.GetEventTags(context.Background(), 234)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventTags) != 1 || int(eventTags[0].ID) != 456 || !eventTags[0].ForceHide || !eventTags[0].IsCarousel {
		t.Fatalf("eventTags=%+v", eventTags)
	}
	comments, err := client.GetComment(context.Background(), 567)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 || int(comments[0].ID) != 567 || int(comments[0].EventID) != 234 {
		t.Fatalf("comments=%+v", comments)
	}
}

func TestGammaIntegerIDFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/markets":
			if r.URL.Query().Get("tag_id") != "12" || r.URL.Query().Get("event_id") != "34" {
				t.Fatalf("market query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[]`))
		case "/series":
			if r.URL.Query().Get("tag_id") != "56" {
				t.Fatalf("series query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[]`))
		case "/comments":
			if r.URL.Query().Get("event_id") != "78" {
				t.Fatalf("comments query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	client := New(Config{Host: srv.URL})
	if _, err := client.GetMarkets(context.Background(), MarketFilterParams{TagID: 12, EventID: 34}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListSeries(context.Background(), SeriesFilterParams{TagID: 56}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetComments(context.Background(), CommentFilterParams{EventID: 78}); err != nil {
		t.Fatal(err)
	}
}

func TestGammaKeysetEndpoints(t *testing.T) {
	yes := true
	no := false
	volumeMin := 10.5
	liquidityMin := 0.25

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/events/keyset":
			q := r.URL.Query()
			if r.Method != http.MethodGet ||
				q.Get("limit") != "50" ||
				q.Get("after_cursor") != "evt-cursor" ||
				q.Get("ascending") != "false" ||
				q.Get("live") != "true" ||
				q.Get("title_search") != "election" ||
				q.Get("liquidity_min") != "0.25" ||
				q.Get("include_chat") != "true" ||
				q.Get("locale") != "en-US" {
				t.Fatalf("events query = %s", r.URL.RawQuery)
			}
			if got := q["id"]; len(got) != 2 || got[0] != "1" || got[1] != "2" {
				t.Fatalf("event ids = %v", got)
			}
			if got := q["tag_id"]; len(got) != 2 || got[0] != "7" || got[1] != "8" {
				t.Fatalf("event tag ids = %v", got)
			}
			if got := q["exclude_tag_id"]; len(got) != 1 || got[0] != "9" {
				t.Fatalf("event exclude tags = %v", got)
			}
			_, _ = w.Write([]byte(`{"events":[{"id":"1","title":"Election"}],"next_cursor":"evt-next"}`))
		case "/markets/keyset":
			q := r.URL.Query()
			if r.Method != http.MethodGet ||
				q.Get("limit") != "25" ||
				q.Get("after_cursor") != "mkt-cursor" ||
				q.Get("ascending") != "true" ||
				q.Get("closed") != "false" ||
				q.Get("volume_num_min") != "10.5" ||
				q.Get("include_tag") != "true" ||
				q.Get("uma_resolution_status") != "resolved" {
				t.Fatalf("markets query = %s", r.URL.RawQuery)
			}
			if got := q["condition_ids"]; len(got) != 2 || got[0] != "0x1" || got[1] != "0x2" {
				t.Fatalf("condition ids = %v", got)
			}
			if got := q["market_maker_address"]; len(got) != 1 || got[0] != "0xmaker" {
				t.Fatalf("market maker addresses = %v", got)
			}
			_, _ = w.Write([]byte(`{"markets":[{"id":"10","question":"Question?"}],"next_cursor":"mkt-next"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	client := New(Config{Host: srv.URL})
	events, err := client.ListEventsKeyset(context.Background(), EventKeysetParams{
		Limit:            50,
		AfterCursor:      "evt-cursor",
		Ascending:        &no,
		IDs:              []int{1, 2},
		Live:             &yes,
		TitleSearch:      "election",
		LiquidityMin:     &liquidityMin,
		TagIDs:           []int{7, 8},
		ExcludeTagIDs:    []int{9},
		IncludeChat:      &yes,
		IncludeBestLines: &yes,
		Locale:           "en-US",
	})
	if err != nil {
		t.Fatal(err)
	}
	if events.NextCursor != "evt-next" || len(events.Events) != 1 || events.Events[0].Title != "Election" {
		t.Fatalf("events=%+v", events)
	}

	markets, err := client.ListMarketsKeyset(context.Background(), MarketKeysetParams{
		Limit:               25,
		AfterCursor:         "mkt-cursor",
		Ascending:           &yes,
		Closed:              &no,
		ConditionIDs:        []string{"0x1", "0x2"},
		MarketMakerAddress:  []string{"0xmaker"},
		VolumeNumMin:        &volumeMin,
		IncludeTag:          &yes,
		UMAResolutionStatus: "resolved",
	})
	if err != nil {
		t.Fatal(err)
	}
	if markets.NextCursor != "mkt-next" || len(markets.Markets) != 1 || markets.Markets[0].Question != "Question?" {
		t.Fatalf("markets=%+v", markets)
	}
}
