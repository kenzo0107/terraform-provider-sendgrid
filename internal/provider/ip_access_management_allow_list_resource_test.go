// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/kenzo0107/sendgrid"
)

const fakeAllowListPath = "/access_settings/whitelist"

// fakeAllowListAPI keeps the allow list in memory. No test here touches a real account.
type fakeAllowListAPI struct {
	mu    sync.Mutex
	rules []*sendgrid.AllowedIP
	ids   int64
	calls []string
}

// fakeAllowListCIDR stores addresses as the real API does: 192.0.2.10 as 192.0.2.10/32, 203.0.*.* as 203.0.0.0/16.
func fakeAllowListCIDR(ip string) string {
	if strings.Contains(ip, "/") {
		return ip
	}
	octets := strings.Split(ip, ".")
	prefix := 32
	for i := len(octets) - 1; i >= 0 && octets[i] == "*"; i-- {
		octets[i] = "0"
		prefix -= 8
	}
	return fmt.Sprintf("%s/%d", strings.Join(octets, "."), prefix)
}

func newFakeAllowListAPI(ips ...string) *fakeAllowListAPI {
	f := &fakeAllowListAPI{}
	for _, ip := range ips {
		f.add(ip)
	}
	return f
}

func (f *fakeAllowListAPI) add(ip string) *sendgrid.AllowedIP {
	ip = fakeAllowListCIDR(ip)
	for _, rule := range f.rules {
		if rule.IP == ip {
			return rule
		}
	}
	f.ids++
	rule := &sendgrid.AllowedIP{ID: f.ids, IP: ip, CreatedAt: 1, UpdatedAt: 1}
	f.rules = append(f.rules, rule)
	return rule
}

func (f *fakeAllowListAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path != fakeAllowListPath {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(sendgrid.OutputGetAllowedIPs{Result: f.rules})
	case http.MethodPost:
		var in sendgrid.InputAddIPsToAllowList
		if !decodeFakeRequest(w, r, &in) {
			return
		}
		var out sendgrid.OutputAddIPsToAllowList
		var sent []string
		for _, ip := range in.IPs {
			sent = append(sent, ip.IP)
			out.Result = append(out.Result, f.add(ip.IP))
		}
		f.calls = append(f.calls, "POST "+strings.Join(sent, ","))
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(out)
	case http.MethodDelete:
		var in sendgrid.InputRemoveIPsFromAllowList
		if !decodeFakeRequest(w, r, &in) {
			return
		}
		f.calls = append(f.calls, fmt.Sprintf("DELETE %v", in.IDs))
		f.rules = slices.DeleteFunc(f.rules, func(rule *sendgrid.AllowedIP) bool {
			return slices.Contains(in.IDs, rule.ID)
		})
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (f *fakeAllowListAPI) ips() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	ips := []string{}
	for _, rule := range f.rules {
		ips = append(ips, rule.IP)
	}
	sort.Strings(ips)
	return ips
}

func (f *fakeAllowListAPI) writes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.calls)
}

func fakeAllowListProviderFactories(t *testing.T, api *fakeAllowListAPI) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	client := sendgrid.New("dummy", sendgrid.OptionBaseURL(srv.URL))
	return map[string]func() (tfprotov6.ProviderServer, error){
		"sendgrid": providerserver.NewProtocol6WithError(&fakeAPIProvider{sendgridProvider: sendgridProvider{version: "test"}, client: client}),
	}
}

func allowListConfig(acknowledge bool, ips ...string) string {
	quoted := make([]string, 0, len(ips))
	for _, ip := range ips {
		quoted = append(quoted, fmt.Sprintf("%q", ip))
	}
	return fmt.Sprintf(`
resource "sendgrid_ip_access_management_allow_list" "test" {
  ips                      = [%s]
  acknowledge_lockout_risk = %t
}
`, strings.Join(quoted, ", "), acknowledge)
}

func expectStrings(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func TestIPAccessManagementAllowListResource_Lifecycle(t *testing.T) {
	api := newFakeAllowListAPI("203.0.113.1")
	resourceName := "sendgrid_ip_access_management_allow_list.test"
	created := allowListConfig(true, "192.0.2.10", "198.51.100.0/24", "10.*.*.*")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAllowListProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: created,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", ipAccessManagementAllowListID),
					resource.TestCheckResourceAttr(resourceName, "ips.#", "3"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ips.*", "198.51.100.0/24"),
					resource.TestCheckResourceAttr(resourceName, "rule_ids.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule_ids.10.*.*.*", "2"),
					func(*terraform.State) error {
						expectStrings(t, "writes after create", api.writes(), []string{"POST 10.*.*.*,192.0.2.10,198.51.100.0/24"})
						return nil
					},
				),
			},
			{
				Config: created,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: allowListConfig(true, "192.0.2.10", "192.0.2.20", "192.0.2.30"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "ips.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule_ids.192.0.2.10", "3"),
					func(*terraform.State) error {
						expectStrings(t, "writes after update", api.writes()[1:], []string{"POST 192.0.2.20,192.0.2.30", "DELETE [2 4]"})
						expectStrings(t, "allow list after update", api.ips(), []string{"192.0.2.10/32", "192.0.2.20/32", "192.0.2.30/32", "203.0.113.1/32"})
						return nil
					},
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "192.0.2.10, 192.0.2.20,192.0.2.30",
				ImportStateVerify: true,
			},
		},
	})

	expectStrings(t, "allow list after destroy", api.ips(), []string{"203.0.113.1/32"})
}

func TestIPAccessManagementAllowListResource_RequiresAcknowledgement(t *testing.T) {
	api := newFakeAllowListAPI()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAllowListProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config:      allowListConfig(false, "192.0.2.10"),
				ExpectError: regexp.MustCompile(`acknowledge_lockout_risk`),
			},
			{
				Config:      strings.Replace(allowListConfig(true, "192.0.2.10"), "acknowledge_lockout_risk = true", "", 1),
				ExpectError: regexp.MustCompile(`acknowledge_lockout_risk`),
			},
		},
	})

	expectStrings(t, "writes", api.writes(), nil)
	expectStrings(t, "allow list", api.ips(), []string{})
}

func TestIPAccessManagementAllowListResource_RejectsInvalidIPs(t *testing.T) {
	api := newFakeAllowListAPI()

	for _, ips := range [][]string{
		{}, {"example.com"}, {"192.0.2.256"}, {"192.0.2.1/33"}, {"2001:db8::1"}, {"192.0.2"},
		{"10.*.5.*"}, {"*.1.2.3"}, {"10.*.*.*/8"}, {"192.168.100.14/24"}, {" 192.0.2.1"},
	} {
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: fakeAllowListProviderFactories(t, api),
			Steps: []resource.TestStep{
				{
					Config:      allowListConfig(true, ips...),
					ExpectError: regexp.MustCompile(`ips`),
				},
			},
		})
	}

	expectStrings(t, "writes", api.writes(), nil)
}

func TestIPAccessManagementAllowListResource_NormalizedIPsHaveNoDiff(t *testing.T) {
	api := newFakeAllowListAPI()
	config := allowListConfig(true, "192.0.2.10", "10.*.*.*", "203.0.*.*", "198.51.100.7/32")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAllowListProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sendgrid_ip_access_management_allow_list.test", "rule_ids.10.*.*.*", "1"),
					resource.TestCheckResourceAttr("sendgrid_ip_access_management_allow_list.test", "rule_ids.192.0.2.10", "2"),
					resource.TestCheckResourceAttr("sendgrid_ip_access_management_allow_list.test", "rule_ids.203.0.*.*", "4"),
					func(*terraform.State) error {
						expectStrings(t, "stored allow list", api.ips(), []string{"10.0.0.0/8", "192.0.2.10/32", "198.51.100.7/32", "203.0.0.0/16"})
						return nil
					},
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})

	expectStrings(t, "allow list after destroy", api.ips(), []string{})
}

func TestIPAccessManagementAllowListResource_RejectsAddressAlreadyOnAllowList(t *testing.T) {
	api := newFakeAllowListAPI("10.0.0.0/8")
	existing := regexp.MustCompile(`"10\.\*\.\*\.\*" is already on the allow list as "10\.0\.0\.0/8"`)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAllowListProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config:      allowListConfig(true, "10.*.*.*", "192.0.2.20"),
				ExpectError: existing,
			},
			{
				Config: allowListConfig(true, "192.0.2.20"),
			},
			{
				Config:      allowListConfig(true, "10.*.*.*", "192.0.2.20", "192.0.2.30"),
				ExpectError: existing,
			},
		},
	})

	expectStrings(t, "writes", api.writes(), []string{"POST 192.0.2.20", "DELETE [2]"})
	expectStrings(t, "allow list after destroy", api.ips(), []string{"10.0.0.0/8"})
}

func TestIPAccessManagementAllowListResource_RestoresRemovedAndKeepsRespelledRule(t *testing.T) {
	api := newFakeAllowListAPI()
	resourceName := "sendgrid_ip_access_management_allow_list.test"
	config := allowListConfig(true, "192.0.2.10", "192.0.2.20")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAllowListProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				PreConfig: func() {
					api.mu.Lock()
					defer api.mu.Unlock()
					api.rules = api.rules[:1]
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule_ids.192.0.2.10", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule_ids.192.0.2.20", "3"),
				),
			},
			{
				Config: allowListConfig(true, "192.0.2.10/32", "192.0.2.20"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule_ids.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule_ids.192.0.2.10/32", "1"),
					func(*terraform.State) error {
						expectStrings(t, "writes after rewriting an IP as /32", api.writes(), []string{"POST 192.0.2.10,192.0.2.20", "POST 192.0.2.20"})
						return nil
					},
				),
			},
		},
	})

	expectStrings(t, "allow list after destroy", api.ips(), []string{})
}

func TestIPAccessManagementAllowListResource_ImportRejectsUnknownIP(t *testing.T) {
	api := newFakeAllowListAPI("192.0.2.10")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAllowListProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config:        allowListConfig(true, "192.0.2.10", "192.0.2.99"),
				ResourceName:  "sendgrid_ip_access_management_allow_list.test",
				ImportState:   true,
				ImportStateId: "192.0.2.10,192.0.2.99",
				ExpectError:   regexp.MustCompile(`"192.0.2.99" is not on the allow list`),
			},
		},
	})

	expectStrings(t, "writes", api.writes(), nil)
}

func TestIPAccessManagementAllowListResource_ImportsEquivalentRanges(t *testing.T) {
	api := newFakeAllowListAPI("10.0.0.0/8", "198.51.100.0/24", "192.0.2.10/32", "203.0.113.1")
	resourceName := "sendgrid_ip_access_management_allow_list.test"
	config := allowListConfig(true, "10.*.*.*", "198.51.100.0/24", "192.0.2.10")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: fakeAllowListProviderFactories(t, api),
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStateId:      "10.*.*.*,198.51.100.0/24,192.0.2.10",
				ImportStatePersist: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule_ids.10.*.*.*", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule_ids.198.51.100.0/24", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule_ids.192.0.2.10", "3"),
				),
			},
		},
	})

	expectStrings(t, "writes", api.writes(), []string{"DELETE [1 2 3]"})
	expectStrings(t, "allow list after destroy", api.ips(), []string{"203.0.113.1/32"})
}

func TestCanonicalAllowListIP(t *testing.T) {
	for ip, want := range map[string]string{
		"192.0.2.10":      "192.0.2.10/32",
		" 192.0.2.10/32 ": "192.0.2.10/32",
		"10.*.*.*":        "10.0.0.0/8",
		"192.168.*.*":     "192.168.0.0/16",
		"203.0.113.*":     "203.0.113.0/24",
		"198.51.100.0/24": "198.51.100.0/24",
	} {
		if got := canonicalAllowListIP(ip); got != want {
			t.Errorf("canonicalAllowListIP(%q) = %q, want %q", ip, got, want)
		}
	}
}

func TestMatchAddedIPs(t *testing.T) {
	rules := func(ips ...string) []*sendgrid.AllowedIP {
		out := []*sendgrid.AllowedIP{}
		for i, ip := range ips {
			out = append(out, &sendgrid.AllowedIP{ID: int64(i + 1), IP: ip})
		}
		return out
	}

	requested := []string{"10.*.*.*", "192.168.*.*", "192.0.2.1", "198.51.100.0/24"}
	got, err := matchAddedIPs(requested, rules("198.51.100.0/24", "192.0.2.1/32", "192.168.0.0/16", "10.0.0.0/8"))
	want := map[string]int64{"198.51.100.0/24": 1, "192.0.2.1": 2, "192.168.*.*": 3, "10.*.*.*": 4}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("matchAddedIPs with a reordered response = %v, %v, want %v", got, err, want)
	}

	got, err = matchAddedIPs([]string{"192.0.2.1", "192.0.2.1/32"}, rules("192.0.2.1/32"))
	if want := map[string]int64{"192.0.2.1": 1, "192.0.2.1/32": 1}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("matchAddedIPs with two spellings of one rule = %v, %v, want %v", got, err, want)
	}

	if _, err := matchAddedIPs([]string{"192.0.2.1", "192.0.2.2"}, rules("192.0.2.1/32")); err == nil {
		t.Error("matchAddedIPs with a missing rule: want an error")
	}
	if _, err := matchAddedIPs([]string{"10.*.*.*", "192.168.*.*"}, rules("10.0.0.0/8", "172.16.0.0/12")); err == nil {
		t.Error("matchAddedIPs with a rule for another range: want an error, not a positional match")
	}
}
