package livefind

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	openapiclient "github.com/xurrent/go-xurrent"
)

// tryNameFilterPages lists collection path with server-side name= (filtering docs) plus
// pagination. If the first response is 400/422, the API likely rejects unknown filters —
// returns filterWorked=false so callers can fall back to a full scan.
func tryNameFilterPages(ctx context.Context, client *openapiclient.APIClient, path, name, fields string, states []string, setState func(q url.Values, state string), appendPage func(body []byte, state string, res *Result, needle string) error, res *Result, needle string) (filterWorked bool, err error) {
	first := true
	for _, st := range states {
		var after string
		for range listPageLimit() {
			q := url.Values{}
			q.Set("per_page", strconv.Itoa(int(listPageSize)))
			q.Set("fields", fields)
			q.Set("name", name)
			setState(q, st)
			if after != "" {
				q.Set("search_after", after)
			}
			body, resp, err := client.GetCollectionJSON(ctx, path, q)
			if err != nil {
				return false, err
			}
			if first && resp != nil && (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity) {
				return false, nil
			}
			first = false
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return false, fmt.Errorf("%s: HTTP %d", path, resp.StatusCode)
			}
			if err := appendPage(body, st, res, needle); err != nil {
				return false, err
			}
			after = nextSearchAfter(resp)
			if after == "" {
				break
			}
		}
	}
	return true, nil
}

func searchTeamName(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	needle := strings.TrimSpace(name)
	ok, err := tryNameFilterPages(ctx, client, "/v1/teams", name, "id,name,disabled", []string{"disabled", "enabled"},
		func(q url.Values, st string) { q.Set("state", st) },
		func(body []byte, st string, res *Result, needle string) error {
			var rows []openapiclient.GetTeams200ResponseInner
			if err := json.Unmarshal(body, &rows); err != nil {
				return err
			}
			for i := range rows {
				n := ""
				if rows[i].Name != nil {
					n = *rows[i].Name
				}
				if !matchFold(n, needle) {
					continue
				}
				res.Matches = append(res.Matches, Row{
					ID:     idStr(rows[i].Id),
					Source: "teams:" + st,
					Label:  n,
					FieldValues: map[string]string{
						"name": n,
					},
				})
			}
			return nil
		}, res, needle)
	if err != nil {
		return err
	}
	if ok && len(res.Matches) > 0 {
		addNote(res, "Used server-side name= filter on GET /v1/teams (per filtering docs).")
		dedupe(&res.Matches)
		return nil
	}
	if ok && len(res.Matches) == 0 {
		addNote(res, "name= filter returned no rows; scanning full team lists for case-insensitive match.")
	} else if !ok {
		addNote(res, "Server did not accept name= on teams collection; using full paginated scan.")
	}
	res.Matches = res.Matches[:0]
	return teamNameScanOnly(ctx, client, name, res)
}

func teamNameScanOnly(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	states := []string{"disabled", "enabled"}
	for _, st := range states {
		var after string
		for range listPageLimit() {
			req := client.TeamsAPI.GetTeams(ctx).State(st).PerPage(listPageSize).Fields("id,name,disabled")
			if after != "" {
				req = req.SearchAfter(after)
			}
			rows, resp, err := req.Execute()
			if err != nil {
				return fmt.Errorf("GetTeams state=%s: %w", st, err)
			}
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("GetTeams state=%s: HTTP %d", st, resp.StatusCode)
			}
			for i := range rows {
				n := ""
				if rows[i].Name != nil {
					n = *rows[i].Name
				}
				if !matchFold(n, name) {
					continue
				}
				res.Matches = append(res.Matches, Row{
					ID:     idStr(rows[i].Id),
					Source: "teams:" + st,
					Label:  n,
					FieldValues: map[string]string{
						"name": n,
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

func searchSiteName(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	needle := strings.TrimSpace(name)
	ok, err := tryNameFilterPages(ctx, client, "/v1/sites", name, "id,name", []string{"disabled", "enabled"},
		func(q url.Values, st string) { q.Set("state", st) },
		func(body []byte, st string, res *Result, needle string) error {
			var rows []openapiclient.GetSites200ResponseInner
			if err := json.Unmarshal(body, &rows); err != nil {
				return err
			}
			for i := range rows {
				n := ""
				if rows[i].Name != nil {
					n = *rows[i].Name
				}
				if !matchFold(n, needle) {
					continue
				}
				res.Matches = append(res.Matches, Row{
					ID:     idStr(rows[i].Id),
					Source: "sites:" + st,
					Label:  n,
					FieldValues: map[string]string{
						"name": n,
					},
				})
			}
			return nil
		}, res, needle)
	if err != nil {
		return err
	}
	if ok && len(res.Matches) > 0 {
		addNote(res, "Used server-side name= filter on GET /v1/sites (per filtering docs).")
		dedupe(&res.Matches)
		return nil
	}
	if ok && len(res.Matches) == 0 {
		addNote(res, "name= filter returned no rows; scanning full site lists for case-insensitive match.")
	} else if !ok {
		addNote(res, "Server did not accept name= on sites collection; using full paginated scan.")
	}
	res.Matches = res.Matches[:0]
	return siteNameScanOnly(ctx, client, name, res)
}

func siteNameScanOnly(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	states := []string{"disabled", "enabled"}
	for _, st := range states {
		var after string
		for range listPageLimit() {
			req := client.SitesAPI.GetSites(ctx).State(st).PerPage(listPageSize).Fields("id,name")
			if after != "" {
				req = req.SearchAfter(after)
			}
			rows, resp, err := req.Execute()
			if err != nil {
				return fmt.Errorf("GetSites state=%s: %w", st, err)
			}
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("GetSites state=%s: HTTP %d", st, resp.StatusCode)
			}
			for i := range rows {
				n := ""
				if rows[i].Name != nil {
					n = *rows[i].Name
				}
				if !matchFold(n, name) {
					continue
				}
				res.Matches = append(res.Matches, Row{
					ID:     idStr(rows[i].Id),
					Source: "sites:" + st,
					Label:  n,
					FieldValues: map[string]string{
						"name": n,
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

func searchServiceName(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	needle := strings.TrimSpace(name)
	ok, err := tryNameFilterPages(ctx, client, "/v1/services", name, "id,name,disabled", []string{"disabled", "enabled"},
		func(q url.Values, st string) { q.Set("state", st) },
		func(body []byte, st string, res *Result, needle string) error {
			var rows []map[string]interface{}
			if err := json.Unmarshal(body, &rows); err != nil {
				return err
			}
			for i := range rows {
				n := mapStr(rows[i], "name")
				if !matchFold(n, needle) {
					continue
				}
				res.Matches = append(res.Matches, Row{
					ID:     mapStr(rows[i], "id"),
					Source: "services:" + st,
					Label:  n,
					FieldValues: map[string]string{
						"name": n,
					},
				})
			}
			return nil
		}, res, needle)
	if err != nil {
		return err
	}
	if ok && len(res.Matches) > 0 {
		addNote(res, "Used server-side name= filter on GET /v1/services (per filtering docs).")
		dedupe(&res.Matches)
		return nil
	}
	if ok && len(res.Matches) == 0 {
		addNote(res, "name= filter returned no rows; scanning full service lists for case-insensitive match.")
	} else if !ok {
		addNote(res, "Server did not accept name= on services collection; using full paginated scan.")
	}
	res.Matches = res.Matches[:0]
	return serviceNameScanOnly(ctx, client, name, res)
}

func serviceNameScanOnly(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	states := []string{"disabled", "enabled"}
	for _, st := range states {
		var after string
		for range listPageLimit() {
			req := client.ServicesAPI.GetServices(ctx).State(st).PerPage(listPageSize).Fields("id,name,disabled")
			if after != "" {
				req = req.SearchAfter(after)
			}
			rows, resp, err := req.Execute()
			if err != nil {
				return fmt.Errorf("GetServices state=%s: %w", st, err)
			}
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("GetServices state=%s: HTTP %d", st, resp.StatusCode)
			}
			for i := range rows {
				n := mapStr(rows[i], "name")
				if !matchFold(n, name) {
					continue
				}
				res.Matches = append(res.Matches, Row{
					ID:     mapStr(rows[i], "id"),
					Source: "services:" + st,
					Label:  n,
					FieldValues: map[string]string{
						"name": n,
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

func searchOrgName(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	needle := strings.TrimSpace(name)
	ok, err := tryNameFilterPages(ctx, client, "/v1/organizations", name, "id,name", []string{"disabled", "enabled"},
		func(q url.Values, st string) { q.Set("state", st) },
		func(body []byte, st string, res *Result, needle string) error {
			var rows []openapiclient.GetOrganizations200ResponseInner
			if err := json.Unmarshal(body, &rows); err != nil {
				return err
			}
			for i := range rows {
				n := ""
				if rows[i].Name != nil {
					n = *rows[i].Name
				}
				if !matchFold(n, needle) {
					continue
				}
				res.Matches = append(res.Matches, Row{
					ID:     idStr(rows[i].Id),
					Source: "organizations:" + st,
					Label:  n,
					FieldValues: map[string]string{
						"name": n,
					},
				})
			}
			return nil
		}, res, needle)
	if err != nil {
		return err
	}
	if ok && len(res.Matches) > 0 {
		addNote(res, "Used server-side name= filter on GET /v1/organizations (per filtering docs).")
		dedupe(&res.Matches)
		return nil
	}
	if ok && len(res.Matches) == 0 {
		addNote(res, "name= filter returned no rows; scanning full organization lists for case-insensitive match.")
	} else if !ok {
		addNote(res, "Server did not accept name= on organizations collection; using full paginated scan.")
	}
	res.Matches = res.Matches[:0]
	return orgNameScanOnly(ctx, client, name, res)
}

func orgNameScanOnly(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	for _, st := range []string{"disabled", "enabled"} {
		var after string
		for range listPageLimit() {
			req := client.OrganizationsAPI.GetOrganizations(ctx).State(st).PerPage(listPageSize).Fields("id,name")
			if after != "" {
				req = req.SearchAfter(after)
			}
			rows, resp, err := req.Execute()
			if err != nil {
				return fmt.Errorf("GetOrganizations state=%s: %w", st, err)
			}
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("GetOrganizations state=%s: HTTP %d", st, resp.StatusCode)
			}
			for i := range rows {
				n := ""
				if rows[i].Name != nil {
					n = *rows[i].Name
				}
				if !matchFold(n, name) {
					continue
				}
				res.Matches = append(res.Matches, Row{
					ID:     idStr(rows[i].Id),
					Source: "organizations:" + st,
					Label:  n,
					FieldValues: map[string]string{
						"name": n,
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

func searchSLAName(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	needle := strings.TrimSpace(name)
	states := []string{"", "active", "inactive"}
	ok, err := tryNameFilterPages(ctx, client, "/v1/slas", name, "id,name", states,
		func(q url.Values, st string) {
			if st != "" {
				q.Set("state", st)
			}
		},
		func(body []byte, st string, res *Result, needle string) error {
			var rows []map[string]interface{}
			if err := json.Unmarshal(body, &rows); err != nil {
				return err
			}
			for _, row := range rows {
				n := ""
				if v, ok := row["name"].(string); ok {
					n = v
				}
				if !matchFold(n, needle) {
					continue
				}
				id := ""
				switch v := row["id"].(type) {
				case float64:
					id = strconv.FormatInt(int64(v), 10)
				case string:
					id = v
				}
				res.Matches = append(res.Matches, Row{
					ID:          id,
					Source:      "slas:" + st,
					Label:       n,
					FieldValues: map[string]string{"name": n},
				})
			}
			return nil
		}, res, needle)
	if err != nil {
		return err
	}
	if ok && len(res.Matches) > 0 {
		addNote(res, "Used server-side name= filter on GET /v1/slas (per filtering docs).")
		dedupe(&res.Matches)
		return nil
	}
	if ok && len(res.Matches) == 0 {
		addNote(res, "name= filter returned no rows; scanning full SLA lists for case-insensitive match.")
	} else if !ok {
		addNote(res, "Server did not accept name= on SLAs collection; using full paginated scan.")
	}
	res.Matches = res.Matches[:0]
	return slaNameScanOnly(ctx, client, name, res)
}

func slaNameScanOnly(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	for _, st := range []string{"", "active", "inactive"} {
		var after string
		for range listPageLimit() {
			req := client.ServiceLevelAgreementsAPI.GetSlas(ctx).PerPage(listPageSize).Fields("id,name")
			if st != "" {
				req = req.State(st)
			}
			if after != "" {
				req = req.SearchAfter(after)
			}
			rows, resp, err := req.Execute()
			if err != nil {
				addNote(res, fmt.Sprintf("GetSlas state=%q skipped: %v", st, err))
				break
			}
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				addNote(res, fmt.Sprintf("GetSlas state=%q: HTTP %d", st, resp.StatusCode))
				break
			}
			for _, row := range rows {
				n := ""
				if v, ok := row["name"].(string); ok {
					n = v
				}
				if !matchFold(n, name) {
					continue
				}
				id := ""
				switch v := row["id"].(type) {
				case float64:
					id = strconv.FormatInt(int64(v), 10)
				case string:
					id = v
				}
				res.Matches = append(res.Matches, Row{
					ID:          id,
					Source:      "slas:" + st,
					Label:       n,
					FieldValues: map[string]string{"name": n},
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

func searchProductName(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	needle := strings.TrimSpace(value)
	first := true
	type pathSpec struct {
		path          string
		source        string
		disabledTyped bool
	}
	paths := []pathSpec{
		{"/v1/products/disabled", "products:disabled", true},
		{"/v1/products/enabled", "products:enabled", false},
	}
	for _, p := range paths {
		var after string
		for range listPageLimit() {
			q := url.Values{}
			q.Set("per_page", strconv.Itoa(int(listPageSize)))
			q.Set("fields", "id,name,disabled")
			q.Set("name", value)
			if after != "" {
				q.Set("search_after", after)
			}
			body, resp, err := client.GetCollectionJSON(ctx, p.path, q)
			if err != nil {
				return err
			}
			if first && resp != nil && (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity) {
				res.Matches = res.Matches[:0]
				addNote(res, "Server did not accept name= on products collection; using full paginated scan.")
				return productNameScanOnly(ctx, client, value, res)
			}
			first = false
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("%s: HTTP %d", p.path, resp.StatusCode)
			}
			if p.disabledTyped {
				if err := appendProductDisabledNameRows(body, res, needle, p.source); err != nil {
					return err
				}
			} else {
				if err := appendProductEnabledNameRows(body, res, needle, p.source); err != nil {
					return err
				}
			}
			after = nextSearchAfter(resp)
			if after == "" {
				break
			}
		}
	}
	if len(res.Matches) > 0 {
		addNote(res, "Used server-side name= filter on GET /v1/products/disabled|enabled (per filtering docs).")
		dedupe(&res.Matches)
		return nil
	}
	addNote(res, "name= filter returned no rows; scanning full product lists for case-insensitive match.")
	res.Matches = res.Matches[:0]
	return productNameScanOnly(ctx, client, value, res)
}

func appendProductDisabledNameRows(body []byte, res *Result, needle, source string) error {
	var rows []openapiclient.GetProducts200ResponseInner
	if err := json.Unmarshal(body, &rows); err != nil {
		return err
	}
	for i := range rows {
		n := ""
		if rows[i].Name != nil {
			n = *rows[i].Name
		}
		if !matchFold(n, needle) {
			continue
		}
		res.Matches = append(res.Matches, productRow(&rows[i], source))
	}
	return nil
}

func appendProductEnabledNameRows(body []byte, res *Result, needle, source string) error {
	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return err
	}
	for i := range rows {
		n := mapStr(rows[i], "name")
		if !matchFold(n, needle) {
			continue
		}
		res.Matches = append(res.Matches, Row{
			ID:     mapStr(rows[i], "id"),
			Source: source,
			Label:  n,
			FieldValues: map[string]string{
				"name": n,
			},
		})
	}
	return nil
}

func productNameScanOnly(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	var afterD string
	for range listPageLimit() {
		req := client.ProductsAPI.GetProductsDisabled(ctx).PerPage(listPageSize).Fields("id,name,disabled")
		if afterD != "" {
			req = req.SearchAfter(afterD)
		}
		dRows, resp, err := req.Execute()
		if err != nil {
			return fmt.Errorf("GetProductsDisabled: %w", err)
		}
		if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("GetProductsDisabled: HTTP %d", resp.StatusCode)
		}
		for i := range dRows {
			n := ""
			if dRows[i].Name != nil {
				n = *dRows[i].Name
			}
			if !matchFold(n, value) {
				continue
			}
			res.Matches = append(res.Matches, productRow(&dRows[i], "products:disabled"))
		}
		afterD = nextSearchAfter(resp)
		if afterD == "" {
			break
		}
	}
	var afterE string
	for range listPageLimit() {
		req := client.ProductsAPI.GetProductsEnabled(ctx).PerPage(listPageSize).Fields("id,name,disabled")
		if afterE != "" {
			req = req.SearchAfter(afterE)
		}
		eRows, resp2, err := req.Execute()
		if err != nil {
			return fmt.Errorf("GetProductsEnabled: %w", err)
		}
		if resp2 != nil && resp2.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("GetProductsEnabled: HTTP %d", resp2.StatusCode)
		}
		for i := range eRows {
			n := mapStr(eRows[i], "name")
			if !matchFold(n, value) {
				continue
			}
			res.Matches = append(res.Matches, Row{
				ID:     mapStr(eRows[i], "id"),
				Source: "products:enabled",
				Label:  n,
				FieldValues: map[string]string{
					"name": n,
				},
			})
		}
		afterE = nextSearchAfter(resp2)
		if afterE == "" {
			break
		}
	}
	dedupe(&res.Matches)
	return nil
}

// appendProductRowsByProductID parses raw product collection JSON and appends rows whose productID matches needle.
func appendProductRowsByProductID(body []byte, res *Result, needle, source string) error {
	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return err
	}
	for i := range rows {
		pid := mapStr(rows[i], "productID")
		if !matchFold(pid, needle) {
			continue
		}
		n := mapStr(rows[i], "name")
		b := mapStr(rows[i], "brand")
		res.Matches = append(res.Matches, Row{
			ID:     mapStr(rows[i], "id"),
			Source: source,
			Label:  n,
			FieldValues: map[string]string{
				"name":      n,
				"brand":     b,
				"productID": pid,
			},
		})
	}
	return nil
}

// searchProductByProductID lists products by productID using server-side productID= when accepted, else scans.
func searchProductByProductID(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	needle := strings.TrimSpace(value)
	paths := []struct {
		path   string
		source string
	}{
		{"/v1/products/disabled", "products:disabled"},
		{"/v1/products/enabled", "products:enabled"},
	}
	first := true
	for _, p := range paths {
		var after string
		for range listPageLimit() {
			q := url.Values{}
			q.Set("per_page", strconv.Itoa(int(listPageSize)))
			q.Set("fields", "id,name,brand,productID,disabled")
			q.Set("productID", value)
			if after != "" {
				q.Set("search_after", after)
			}
			body, resp, err := client.GetCollectionJSON(ctx, p.path, q)
			if err != nil {
				return err
			}
			if first && resp != nil && (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity) {
				res.Matches = res.Matches[:0]
				addNote(res, "Server did not accept productID= on products collection; scanning without filter.")
				return productProductIDScanOnly(ctx, client, value, res)
			}
			first = false
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("%s: HTTP %d", p.path, resp.StatusCode)
			}
			if err := appendProductRowsByProductID(body, res, needle, p.source); err != nil {
				return err
			}
			after = nextSearchAfter(resp)
			if after == "" {
				break
			}
		}
	}
	if len(res.Matches) > 0 {
		addNote(res, "Used server-side productID= filter on GET /v1/products/disabled|enabled.")
		dedupe(&res.Matches)
		return nil
	}
	addNote(res, "productID= filter returned no rows; scanning lists for case-insensitive productID match.")
	res.Matches = res.Matches[:0]
	return productProductIDScanOnly(ctx, client, value, res)
}

func productProductIDScanOnly(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	needle := strings.TrimSpace(value)
	var afterD string
	for range listPageLimit() {
		q := url.Values{}
		q.Set("per_page", strconv.Itoa(int(listPageSize)))
		q.Set("fields", "id,name,brand,productID,disabled")
		if afterD != "" {
			q.Set("search_after", afterD)
		}
		body, resp, err := client.GetCollectionJSON(ctx, "/v1/products/disabled", q)
		if err != nil {
			return err
		}
		if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("GET /v1/products/disabled: HTTP %d", resp.StatusCode)
		}
		if err := appendProductRowsByProductID(body, res, needle, "products:disabled"); err != nil {
			return err
		}
		afterD = nextSearchAfter(resp)
		if afterD == "" {
			break
		}
	}
	var afterE string
	for range listPageLimit() {
		req := client.ProductsAPI.GetProductsEnabled(ctx).PerPage(listPageSize).Fields("id,name,brand,productID,disabled")
		if afterE != "" {
			req = req.SearchAfter(afterE)
		}
		eRows, resp2, err := req.Execute()
		if err != nil {
			return fmt.Errorf("GetProductsEnabled: %w", err)
		}
		if resp2 != nil && resp2.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("GetProductsEnabled: HTTP %d", resp2.StatusCode)
		}
		for i := range eRows {
			pid := mapStr(eRows[i], "productID")
			if !matchFold(pid, needle) {
				continue
			}
			n := mapStr(eRows[i], "name")
			b := mapStr(eRows[i], "brand")
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
		afterE = nextSearchAfter(resp2)
		if afterE == "" {
			break
		}
	}
	dedupe(&res.Matches)
	return nil
}
