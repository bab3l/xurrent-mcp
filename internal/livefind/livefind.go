// Package livefind runs read-only collection GETs to locate rows that may hold a duplicate identifier.
package livefind

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	openapiclient "github.com/bab3l/go-xurrent"
	"github.com/bab3l/go-xurrent/pkg/collisionhints"
)

const listPageSize int32 = 100

// Row is one possible collision match (subset of API fields).
type Row struct {
	ID          string            `json:"id"`
	Source      string            `json:"source"`
	Label       string            `json:"label,omitempty"`
	FieldValues map[string]string `json:"field_values,omitempty"`
}

// Result is the outcome of a live search.
type Result struct {
	Matches []Row    `json:"matches"`
	Notes   []string `json:"notes"`
}

// ClientFromEnv builds an API client using XURRENT_TOKEN and XURRENT_ACCOUNT.
// Optional: XURRENT_API_BASE overrides the default https://api.xurrent.com server URL.
func ClientFromEnv() (*openapiclient.APIClient, error) {
	token := strings.TrimSpace(os.Getenv("XURRENT_TOKEN"))
	account := strings.TrimSpace(os.Getenv("XURRENT_ACCOUNT"))
	if token == "" || account == "" {
		return nil, fmt.Errorf("livefind: set XURRENT_TOKEN and XURRENT_ACCOUNT in the MCP process environment")
	}
	cfg := openapiclient.NewConfiguration()
	cfg.AddDefaultHeader("Authorization", "Bearer "+token)
	cfg.AddDefaultHeader("X-4me-Account", account)
	if base := strings.TrimSpace(os.Getenv("XURRENT_API_BASE")); base != "" {
		cfg.Servers = openapiclient.ServerConfigurations{{URL: strings.TrimRight(base, "/"), Description: "env"}}
	}
	return openapiclient.NewAPIClient(cfg), nil
}

// Search queries list endpoints (with cursor pagination) and returns rows whose field matches value (case-insensitive).
// valueSecondary is used for composite keys: person (source + sourceID) → value=source, valueSecondary=sourceID;
// product (brand + productID) → value=brand, valueSecondary=productID.
func Search(ctx context.Context, client *openapiclient.APIClient, kind collisionhints.Kind, field, value, valueSecondary string) (*Result, error) {
	field = strings.TrimSpace(strings.ToLower(field))
	if field == "" {
		field = "name"
	}
	value = strings.TrimSpace(value)
	valueSecondary = strings.TrimSpace(valueSecondary)
	if value == "" {
		return nil, fmt.Errorf("livefind: value is required")
	}

	res := &Result{}
	var err error
	switch kind {
	case collisionhints.KindTeam:
		if valueSecondary != "" {
			return nil, fmt.Errorf("livefind: value_secondary is not used for teams")
		}
		err = searchTeamName(ctx, client, value, res)
	case collisionhints.KindSite:
		if valueSecondary != "" {
			return nil, fmt.Errorf("livefind: value_secondary is not used for sites")
		}
		err = searchSiteName(ctx, client, value, res)
	case collisionhints.KindService:
		if valueSecondary != "" {
			return nil, fmt.Errorf("livefind: value_secondary is not used for services")
		}
		err = searchServiceName(ctx, client, value, res)
	case collisionhints.KindPerson:
		err = searchPerson(ctx, client, field, value, valueSecondary, res)
	case collisionhints.KindProduct:
		err = searchProduct(ctx, client, field, value, valueSecondary, res)
	case collisionhints.KindOrganization:
		if valueSecondary != "" {
			return nil, fmt.Errorf("livefind: value_secondary is not used for organizations")
		}
		err = searchOrgName(ctx, client, value, res)
	case collisionhints.KindCI:
		if valueSecondary != "" {
			return nil, fmt.Errorf("livefind: value_secondary is not used for configuration items")
		}
		err = searchCI(ctx, client, field, value, res)
	case collisionhints.KindSLA:
		if valueSecondary != "" {
			return nil, fmt.Errorf("livefind: value_secondary is not used for SLAs")
		}
		err = searchSLAName(ctx, client, value, res)
	default:
		err = fmt.Errorf("livefind: unsupported entity %s", kind)
	}
	if err != nil {
		return nil, err
	}
	res.Notes = append(res.Notes, "Pagination: up to "+strconv.Itoa(listPageLimit())+" pages × "+strconv.Itoa(int(listPageSize))+" rows per list (Link rel=next → search_after; cap via XURRENT_LIVEFIND_MAX_PAGES).")
	return res, nil
}

func addNote(res *Result, msg string) {
	res.Notes = append(res.Notes, msg)
}

func idStr(id *float32) string {
	if id == nil {
		return ""
	}
	return strconv.FormatInt(int64(*id), 10)
}

func matchFold(hay, needle string) bool {
	return strings.EqualFold(strings.TrimSpace(hay), strings.TrimSpace(needle))
}

// mapStr reads a string-ish value from a generic JSON-decoded map.
func mapStr(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func dedupe(rows *[]Row) {
	seen := make(map[string]struct{})
	out := (*rows)[:0]
	for _, r := range *rows {
		k := r.Source + ":" + r.ID
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, r)
	}
	*rows = out
}

func searchPerson(ctx context.Context, client *openapiclient.APIClient, field, value, valueSecondary string, res *Result) error {
	switch field {
	case "source_sourceid", "source+sourceid", "source_source_id":
		if valueSecondary == "" {
			return fmt.Errorf("livefind: for person composite use field source_sourceid with value=source and value_secondary=sourceID")
		}
		return searchPersonSourceSourceID(ctx, client, value, valueSecondary, res)
	case "primary_email", "email":
		if valueSecondary != "" {
			return fmt.Errorf("livefind: value_secondary is not used for primary_email search")
		}
		return searchPersonPrimaryEmail(ctx, client, value, res)
	case "sourceid", "source_id":
		if valueSecondary != "" {
			return fmt.Errorf("livefind: use field source_sourceid for source+sourceID (value=source, value_secondary=sourceID)")
		}
		return searchPersonSourceSourceID(ctx, client, "", value, res)
	case "name":
		if valueSecondary != "" {
			return fmt.Errorf("livefind: value_secondary is not used for person name hint")
		}
		return searchPersonName(ctx, client, value, res)
	default:
		return fmt.Errorf("livefind: unsupported person field %q", field)
	}
}

func searchPersonSourceSourceID(ctx context.Context, client *openapiclient.APIClient, sourceHint, sourceID string, res *Result) error {
	addNote(res, "People list responses do not include `source`; matches use sourceID only. Confirm `source` with GET /v1/people/{id} if needed.")
	if strings.TrimSpace(sourceHint) != "" {
		addNote(res, fmt.Sprintf("Expected source %q is not applied as a list filter (field not present on collection rows).", sourceHint))
	}
	needle := strings.TrimSpace(sourceID)
	var afterDis string
	for range listPageLimit() {
		req := client.PeopleAPI.GetPeopleDisabled(ctx).PerPage(listPageSize).Fields("id,name,primary_email,sourceID")
		if afterDis != "" {
			req = req.SearchAfter(afterDis)
		}
		dis, resp, err := req.Execute()
		if err != nil {
			return fmt.Errorf("GetPeopleDisabled: %w", err)
		}
		if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("GetPeopleDisabled: HTTP %d", resp.StatusCode)
		}
		for i := range dis {
			sid := dis[i].GetSourceID()
			if !matchFold(sid, needle) {
				continue
			}
			n := ""
			if dis[i].Name != nil {
				n = *dis[i].Name
			}
			em := ""
			if dis[i].PrimaryEmail != nil {
				em = *dis[i].PrimaryEmail
			}
			res.Matches = append(res.Matches, Row{
				ID:     idStr(dis[i].Id),
				Source: "people:disabled",
				Label:  n,
				FieldValues: map[string]string{
					"sourceID":      sid,
					"name":          n,
					"primary_email": em,
				},
			})
		}
		afterDis = nextSearchAfter(resp)
		if afterDis == "" {
			break
		}
	}
	var afterEn string
	for range listPageLimit() {
		req := client.PeopleAPI.GetPeopleEnabled(ctx).PerPage(listPageSize).Fields("id,name,primary_email,sourceID")
		if afterEn != "" {
			req = req.SearchAfter(afterEn)
		}
		en, resp2, err := req.Execute()
		if err != nil {
			return fmt.Errorf("GetPeopleEnabled: %w", err)
		}
		if resp2 != nil && resp2.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("GetPeopleEnabled: HTTP %d", resp2.StatusCode)
		}
		for i := range en {
			sid := en[i].GetSourceID()
			if !matchFold(sid, needle) {
				continue
			}
			n := ""
			if en[i].Name != nil {
				n = *en[i].Name
			}
			em := ""
			if en[i].PrimaryEmail != nil {
				em = *en[i].PrimaryEmail
			}
			res.Matches = append(res.Matches, Row{
				ID:     idStr(en[i].Id),
				Source: "people:enabled",
				Label:  n,
				FieldValues: map[string]string{
					"sourceID":      sid,
					"name":          n,
					"primary_email": em,
				},
			})
		}
		afterEn = nextSearchAfter(resp2)
		if afterEn == "" {
			break
		}
	}
	dedupe(&res.Matches)
	return nil
}

func searchProduct(ctx context.Context, client *openapiclient.APIClient, field, value, valueSecondary string, res *Result) error {
	switch field {
	case "brand_productid", "brand+productid", "brand_product_id":
		if valueSecondary == "" {
			return fmt.Errorf("livefind: for product composite use field brand_productid with value=brand and value_secondary=productID")
		}
		return searchProductBrandProductID(ctx, client, value, valueSecondary, res)
	case "name":
		if valueSecondary != "" {
			return fmt.Errorf("livefind: value_secondary is not used for product name search")
		}
		return searchProductName(ctx, client, value, res)
	case "productid", "product_id":
		if valueSecondary != "" {
			return fmt.Errorf("livefind: use field brand_productid for brand+productID (value=brand, value_secondary=productID)")
		}
		return searchProductByProductID(ctx, client, value, res)
	default:
		return fmt.Errorf("livefind: unsupported product field %q", field)
	}
}

func searchProductBrandProductID(ctx context.Context, client *openapiclient.APIClient, brand, productID string, res *Result) error {
	addNote(res, "Composite brand+productID search scans enabled products only; disabled products are not included in this scan.")
	bNeedle := strings.TrimSpace(brand)
	pNeedle := strings.TrimSpace(productID)
	var after string
	for range listPageLimit() {
		req := client.ProductsAPI.GetProductsEnabled(ctx).PerPage(listPageSize).Fields("id,name,brand,productID,disabled")
		if after != "" {
			req = req.SearchAfter(after)
		}
		eRows, resp, err := req.Execute()
		if err != nil {
			return fmt.Errorf("GetProductsEnabled: %w", err)
		}
		if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("GetProductsEnabled: HTTP %d", resp.StatusCode)
		}
		for i := range eRows {
			b := mapStr(eRows[i], "brand")
			pid := mapStr(eRows[i], "productID")
			if !matchFold(b, bNeedle) || !matchFold(pid, pNeedle) {
				continue
			}
			n := mapStr(eRows[i], "name")
			res.Matches = append(res.Matches, Row{
				ID:     mapStr(eRows[i], "id"),
				Source: "products:enabled",
				Label:  n,
				FieldValues: map[string]string{
					"name":      n,
					"brand":     b,
					"productID": pid,
				},
			})
		}
		after = nextSearchAfter(resp)
		if after == "" {
			break
		}
	}
	dedupe(&res.Matches)
	return nil
}

func productRow(p *openapiclient.GetProducts200ResponseInner, source string) Row {
	n := ""
	if p.Name != nil {
		n = *p.Name
	}
	return Row{
		ID:     idStr(p.Id),
		Source: source,
		Label:  n,
		FieldValues: map[string]string{
			"name": n,
		},
	}
}

func searchCI(ctx context.Context, client *openapiclient.APIClient, field, value string, res *Result) error {
	if field != "serial_nr" && field != "serial" {
		return fmt.Errorf("livefind: CI search supports field serial_nr only for now")
	}
	return searchCISerial(ctx, client, value, res)
}
