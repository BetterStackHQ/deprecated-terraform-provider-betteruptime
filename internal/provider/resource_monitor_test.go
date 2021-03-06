package provider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestResourceMonitor(t *testing.T) {
	var data atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("Received " + r.Method + " " + r.RequestURI)

		if r.Header.Get("Authorization") != "Bearer foo" {
			t.Fatal("Not authorized: " + r.Header.Get("Authorization"))
		}

		prefix := "/api/v2/monitors"
		id := "1"

		switch {
		case r.Method == http.MethodPost && r.RequestURI == prefix:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			data.Store(body)
			w.WriteHeader(http.StatusCreated)
			// Inject pronounceable_name.
			computed := make(map[string]interface{})
			if err := json.Unmarshal(body, &computed); err != nil {
				t.Fatal(err)
			}
			computed["pronounceable_name"] = "computed_by_betteruptime"
			body, err = json.Marshal(computed)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"data":{"id":%q,"attributes":%s}}`, id, body)))
		case r.Method == http.MethodGet && r.RequestURI == prefix+"/"+id:
			_, _ = w.Write([]byte(fmt.Sprintf(`{"data":{"id":%q,"attributes":%s}}`, id, data.Load().([]byte))))
		case r.Method == http.MethodPatch && r.RequestURI == prefix+"/"+id:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			patch := make(map[string]interface{})
			if err = json.Unmarshal(data.Load().([]byte), &patch); err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(body, &patch); err != nil {
				t.Fatal(err)
			}
			patched, err := json.Marshal(patch)
			if err != nil {
				t.Fatal(err)
			}
			data.Store(patched)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"data":{"id":%q,"attributes":%s}}`, id, patched)))
		case r.Method == http.MethodDelete && r.RequestURI == prefix+"/"+id:
			w.WriteHeader(http.StatusNoContent)
			data.Store([]byte(nil))
		default:
			t.Fatal("Unexpected " + r.Method + " " + r.RequestURI)
		}
	}))
	defer server.Close()

	var url = "http://example.com"
	var monitorType = "status"

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"betteruptime": func() (*schema.Provider, error) {
				return New(WithURL(server.URL)), nil
			},
		},
		Steps: []resource.TestStep{
			// Step 1 - create.
			{
				Config: fmt.Sprintf(`
				provider "betteruptime" {
					api_token = "foo"
				}

				resource "betteruptime_monitor" "this" {
					url          = "%s"
					monitor_type = "%s"
					paused       = true
					regions      = ["us", "eu"]
				}
				`, url, monitorType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("betteruptime_monitor.this", "id"),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "url", url),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "monitor_type", monitorType),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "paused", "true"),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "pronounceable_name", "computed_by_betteruptime"),
				),
			},
			// Step 2 - update.
			{
				Config: fmt.Sprintf(`
				provider "betteruptime" {
					api_token = "foo"
				}

				resource "betteruptime_monitor" "this" {
					url                = "%s"
					monitor_type       = "%s"
					pronounceable_name = "override"
				}
				`, url, monitorType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("betteruptime_monitor.this", "id"),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "url", url),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "monitor_type", monitorType),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "paused", "false"),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "pronounceable_name", "override"),
				),
			},
			// Step 3 - update (but preserve pronounceable_name).
			{
				Config: fmt.Sprintf(`
				provider "betteruptime" {
					api_token = "foo"
				}

				resource "betteruptime_monitor" "this" {
					url          = "%s"
					monitor_type = "%s"
					http_method  = "POST"
				}
				`, url, monitorType),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("betteruptime_monitor.this", "id"),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "url", url),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "monitor_type", monitorType),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "http_method", "POST"),
					resource.TestCheckResourceAttr("betteruptime_monitor.this", "pronounceable_name", "override"),
				),
			},
			// Step 4 - make no changes, check plan is empty.
			{
				Config: fmt.Sprintf(`
				provider "betteruptime" {
					api_token = "foo"
				}

				resource "betteruptime_monitor" "this" {
					url          = "%s"
					monitor_type = "%s"
					http_method  = "POST"
				}
				`, url, monitorType),
				PlanOnly: true,
			},
			// Step 5 - destroy.
			{
				ResourceName:      "betteruptime_monitor.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
