## 0.1.0 (Unreleased)

FEATURES:

* **New Resource:** `sendgrid_bounce_settings` - Manage bounce settings for your SendGrid account, including soft bounce purge configuration
* **New Data Source:** `sendgrid_bounce_settings` - Retrieve current bounce settings from your SendGrid account

IMPROVEMENTS:

* **Bounce Settings API Migration:** Moved bounce settings API implementation from terraform provider to sendgrid library for better maintainability and consistency
