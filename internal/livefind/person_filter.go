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

func searchPersonName(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	needle := strings.TrimSpace(value)
	first := true
	paths := []struct {
		path          string
		source        string
		disabledTyped bool
	}{
		{"/v1/people/disabled", "people:disabled", true},
		{"/v1/people/enabled", "people:enabled", false},
	}
	for _, p := range paths {
		var after string
		for range listPageLimit() {
			q := url.Values{}
			q.Set("per_page", strconv.Itoa(int(listPageSize)))
			q.Set("fields", "id,name,primary_email")
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
				addNote(res, "Server did not accept name= on people collection; using full paginated scan.")
				return personNameScanOnly(ctx, client, value, res)
			}
			first = false
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("%s: HTTP %d", p.path, resp.StatusCode)
			}
			if p.disabledTyped {
				if err := appendPersonDisabledNameRows(body, res, needle, p.source); err != nil {
					return err
				}
			} else {
				if err := appendPersonEnabledNameRows(body, res, needle, p.source); err != nil {
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
		addNote(res, "Used server-side name= filter on GET /v1/people/disabled|enabled (per filtering docs).")
		dedupe(&res.Matches)
		return nil
	}
	addNote(res, "name= filter returned no rows; scanning full people lists for case-insensitive match.")
	res.Matches = res.Matches[:0]
	return personNameScanOnly(ctx, client, value, res)
}

func appendPersonDisabledNameRows(body []byte, res *Result, needle, source string) error {
	var rows []openapiclient.GetPeopleDisabled200ResponseInner
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
		res.Matches = append(res.Matches, personRowDisabled(&rows[i], source))
	}
	return nil
}

func appendPersonEnabledNameRows(body []byte, res *Result, needle, source string) error {
	var rows []openapiclient.GetPeople200ResponseInner
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
		res.Matches = append(res.Matches, personRowEnabled(&rows[i], source))
	}
	return nil
}

func personRowDisabled(p *openapiclient.GetPeopleDisabled200ResponseInner, source string) Row {
	n := ""
	if p.Name != nil {
		n = *p.Name
	}
	em := ""
	if p.PrimaryEmail != nil {
		em = *p.PrimaryEmail
	}
	return Row{
		ID:     idStr(p.Id),
		Source: source,
		Label:  n,
		FieldValues: map[string]string{
			"name":          n,
			"primary_email": em,
		},
	}
}

func personRowEnabled(p *openapiclient.GetPeople200ResponseInner, source string) Row {
	n := ""
	if p.Name != nil {
		n = *p.Name
	}
	em := ""
	if p.PrimaryEmail != nil {
		em = *p.PrimaryEmail
	}
	return Row{
		ID:     idStr(p.Id),
		Source: source,
		Label:  n,
		FieldValues: map[string]string{
			"name":          n,
			"primary_email": em,
		},
	}
}

func personNameScanOnly(ctx context.Context, client *openapiclient.APIClient, name string, res *Result) error {
	var afterDis string
	for range listPageLimit() {
		req := client.PeopleAPI.GetPeopleDisabled(ctx).PerPage(listPageSize).Fields("id,name,primary_email")
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
			n := ""
			if dis[i].Name != nil {
				n = *dis[i].Name
			}
			if !matchFold(n, name) {
				continue
			}
			res.Matches = append(res.Matches, personRowDisabled(&dis[i], "people:disabled"))
		}
		afterDis = nextSearchAfter(resp)
		if afterDis == "" {
			break
		}
	}
	var afterEn string
	for range listPageLimit() {
		req := client.PeopleAPI.GetPeopleEnabled(ctx).PerPage(listPageSize).Fields("id,name,primary_email")
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
			n := ""
			if en[i].Name != nil {
				n = *en[i].Name
			}
			if !matchFold(n, name) {
				continue
			}
			res.Matches = append(res.Matches, personRowEnabled(&en[i], "people:enabled"))
		}
		afterEn = nextSearchAfter(resp2)
		if afterEn == "" {
			break
		}
	}
	dedupe(&res.Matches)
	return nil
}

func searchPersonPrimaryEmail(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	needle := strings.TrimSpace(value)
	first := true
	paths := []struct {
		path          string
		source        string
		disabledTyped bool
	}{
		{"/v1/people/disabled", "people:disabled", true},
		{"/v1/people/enabled", "people:enabled", false},
	}
	for _, p := range paths {
		var after string
		for range listPageLimit() {
			q := url.Values{}
			q.Set("per_page", strconv.Itoa(int(listPageSize)))
			q.Set("fields", "id,name,primary_email")
			q.Set("primary_email", value)
			if after != "" {
				q.Set("search_after", after)
			}
			body, resp, err := client.GetCollectionJSON(ctx, p.path, q)
			if err != nil {
				return err
			}
			if first && resp != nil && (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity) {
				res.Matches = res.Matches[:0]
				addNote(res, "Server did not accept primary_email= on people collection; using full paginated scan.")
				return personEmailScanOnly(ctx, client, value, res)
			}
			first = false
			if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("%s: HTTP %d", p.path, resp.StatusCode)
			}
			if p.disabledTyped {
				if err := appendPersonDisabledEmailRows(body, res, needle, p.source); err != nil {
					return err
				}
			} else {
				if err := appendPersonEnabledEmailRows(body, res, needle, p.source); err != nil {
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
		addNote(res, "Used server-side primary_email= filter on GET /v1/people/disabled|enabled (per filtering docs).")
		dedupe(&res.Matches)
		return nil
	}
	addNote(res, "primary_email= filter returned no rows; scanning full people lists for case-insensitive match.")
	res.Matches = res.Matches[:0]
	return personEmailScanOnly(ctx, client, value, res)
}

func appendPersonDisabledEmailRows(body []byte, res *Result, needle, source string) error {
	var rows []openapiclient.GetPeopleDisabled200ResponseInner
	if err := json.Unmarshal(body, &rows); err != nil {
		return err
	}
	for i := range rows {
		em := ""
		if rows[i].PrimaryEmail != nil {
			em = *rows[i].PrimaryEmail
		}
		if !matchFold(em, needle) {
			continue
		}
		res.Matches = append(res.Matches, personRowDisabled(&rows[i], source))
	}
	return nil
}

func appendPersonEnabledEmailRows(body []byte, res *Result, needle, source string) error {
	var rows []openapiclient.GetPeople200ResponseInner
	if err := json.Unmarshal(body, &rows); err != nil {
		return err
	}
	for i := range rows {
		em := ""
		if rows[i].PrimaryEmail != nil {
			em = *rows[i].PrimaryEmail
		}
		if !matchFold(em, needle) {
			continue
		}
		res.Matches = append(res.Matches, personRowEnabled(&rows[i], source))
	}
	return nil
}

func personEmailScanOnly(ctx context.Context, client *openapiclient.APIClient, value string, res *Result) error {
	var afterDis string
	for range listPageLimit() {
		req := client.PeopleAPI.GetPeopleDisabled(ctx).PerPage(listPageSize).Fields("id,name,primary_email")
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
			em := ""
			if dis[i].PrimaryEmail != nil {
				em = *dis[i].PrimaryEmail
			}
			if !matchFold(em, value) {
				continue
			}
			res.Matches = append(res.Matches, personRowDisabled(&dis[i], "people:disabled"))
		}
		afterDis = nextSearchAfter(resp)
		if afterDis == "" {
			break
		}
	}
	var afterEn string
	for range listPageLimit() {
		req := client.PeopleAPI.GetPeopleEnabled(ctx).PerPage(listPageSize).Fields("id,name,primary_email")
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
			em := ""
			if en[i].PrimaryEmail != nil {
				em = *en[i].PrimaryEmail
			}
			if !matchFold(em, value) {
				continue
			}
			res.Matches = append(res.Matches, personRowEnabled(&en[i], "people:enabled"))
		}
		afterEn = nextSearchAfter(resp2)
		if afterEn == "" {
			break
		}
	}
	dedupe(&res.Matches)
	return nil
}
