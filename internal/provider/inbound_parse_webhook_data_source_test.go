package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccInboundParseWebhookDataSource(t *testing.T) {
	hostname := os.Getenv("INBOUND_PARSE_WEBHOOK_HOSTNAME")
	if hostname == "" {
		t.Skip()
	}

	resourceName := "data.sendgrid_inbound_parse_webhook.test"
	url := fmt.Sprintf("https://test-acc-%s.com", acctest.RandString(16))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testInboundParseWebhookDataSourceConfig(hostname, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname),
					resource.TestCheckResourceAttr(resourceName, "url", url),
					resource.TestCheckResourceAttr(resourceName, "spam_check", "false"),
					resource.TestCheckResourceAttr(resourceName, "send_raw", "false"),
				),
			},
		},
	})
}

func testInboundParseWebhookDataSourceConfig(hostname, url string) string {
	return fmt.Sprintf(`
resource "sendgrid_inbound_parse_webhook" "test" {
	hostname = "%s"
	url = "%s"
}

data "sendgrid_inbound_parse_webhook" "test" {
	hostname = sendgrid_inbound_parse_webhook.test.hostname
}
`, hostname, url)
}
