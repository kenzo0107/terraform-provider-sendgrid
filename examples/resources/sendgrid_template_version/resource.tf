resource "sendgrid_template" "example" {
  name       = "dummy"
  generation = "dynamic"
}

resource "sendgrid_template_version" "example" {
  template_id = sendgrid_template.example.id
  name        = "dummy"
  active      = 1
  test_data = jsonencode({
    "name" : "dummy"
  })
  html_content = "<%body%>"
}
