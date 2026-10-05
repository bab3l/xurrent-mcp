package livefind

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	openapiclient "github.com/bab3l/go-xurrent"
)

func searchCISerial(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	needle := strings.TrimSpace(value)
	states := []string{"", "active", "archived", "trash"}
	first := true
	seen := make(map[string]struct{})
	for _, st := range states {
		var after string
		for range listPageLimit() {
			q := url.Values{}
			q.Set("per_page", strconv.Itoa(int(listPageSize)))
			q.Set("fields", "id,label,serial_nr,status,state")
			q.Set("serial_nr", value)
			if st != "" {
				q.Set("state", st)
			}
			if after != "" {
				q.Set("search_after", after)
			}
			body, resp, err := client.GetCollectionJSON(ctx, "/v1/cis", q)
			if err != nil {
				return err
			}
			if first && resp != nil && (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity) {
				res.Matches = res.Matches[:0]
				addNote(res, "Server did not accept serial_nr= on GET /v1/cis; using full paginated scan.")
				return ciSerialScanOnly(ctx, client, needle, res)
			}
			first = false
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				addNote(res, fmt.Sprintf("GetCis serial filter state=%q: HTTP %d", st, resp.StatusCode))
				break
			}
			if err := appendCISerialRows(body, res, needle, st, seen); err != nil {
				return err
			}
			after = nextSearchAfter(resp)
			if after == "" {
				break
			}
		}
	}
	if len(res.Matches) > 0 {
		addNote(res, "Used server-side serial_nr= filter on GET /v1/cis.")
		dedupe(&res.Matches)
		return nil
	}
	addNote(res, "serial_nr= filter returned no rows; scanning full CI lists.")
	res.Matches = res.Matches[:0]
	return ciSerialScanOnly(ctx, client, needle, res)
}

func appendCISerialRows(body []byte, res *Result, needle, st string, seen map[string]struct{}) error {
	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return err
	}
	for _, row := range rows {
		sn, _ := row["serial_nr"].(string)
		if sn == "" {
			continue
		}
		if !matchFold(sn, needle) {
			continue
		}
		id := ""
		switch v := row["id"].(type) {
		case float64:
			id = strconv.FormatInt(int64(v), 10)
		case string:
			id = v
		}
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		lbl, _ := row["label"].(string)
		res.Matches = append(res.Matches, Row{
			ID:     id,
			Source: "cis:" + st,
			Label:  lbl,
			FieldValues: map[string]string{
				"serial_nr": sn,
			},
		})
	}
	return nil
}

func ciSerialScanOnly(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	states := []string{"", "active", "archived", "trash"}
	seen := make(map[string]struct{})
	for _, st := range states {
		var after string
		for range listPageLimit() {
			req := client.ConfigurationItemsAPI.GetCis(ctx).PerPage(listPageSize).Fields("id,label,serial_nr,status,state")
			if st != "" {
				req = req.State(st)
			}
			if after != "" {
				req = req.SearchAfter(after)
			}
			rows, resp, err := req.Execute()
			if err != nil {
				addNote(res, fmt.Sprintf("GetCis state=%q skipped: %v", st, err))
				break
			}
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				addNote(res, fmt.Sprintf("GetCis state=%q: HTTP %d", st, resp.StatusCode))
				break
			}
			for _, row := range rows {
				sn, _ := row["serial_nr"].(string)
				if sn == "" {
					continue
				}
				if !matchFold(sn, value) {
					continue
				}
				id := ""
				switch v := row["id"].(type) {
				case float64:
					id = strconv.FormatInt(int64(v), 10)
				case string:
					id = v
				}
				if id == "" {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				lbl, _ := row["label"].(string)
				res.Matches = append(res.Matches, Row{
					ID:     id,
					Source: "cis:" + st,
					Label:  lbl,
					FieldValues: map[string]string{
						"serial_nr": sn,
					},
				})
			}
			after = nextSearchAfter(resp)
			if after == "" {
				break
			}
		}
	}
	dedupe(&res.Matches)
	return nil
}
