## 0.1.0 (Unreleased)

FEATURES:

* **New Resource:** `sendgrid_google_analytics_settings` - Manage the account-wide Google Analytics tracking setting
* **New Data Source:** `sendgrid_google_analytics_settings` - Retrieve the current Google Analytics tracking setting
* **New Resource:** `sendgrid_subscription_tracking_settings` - Manage the account-wide subscription tracking setting
* **New Data Source:** `sendgrid_subscription_tracking_settings` - Retrieve the current subscription tracking setting
* **New Resource:** `sendgrid_open_tracking_settings` - Manage the account-wide open tracking setting
* **New Data Source:** `sendgrid_open_tracking_settings` - Retrieve the current open tracking setting
* **New Resource:** `sendgrid_bounce_settings` - Manage bounce settings for your SendGrid account, including soft bounce purge configuration
* **New Data Source:** `sendgrid_bounce_settings` - Retrieve current bounce settings from your SendGrid account

IMPROVEMENTS:

* **Bounce Settings API Migration:** Moved bounce settings API implementation from terraform provider to sendgrid library for better maintainability and consistency
