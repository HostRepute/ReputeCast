# ReputeCast

[ReputeCast](https://reputecast.com) is the open-source, self-hosted newsletter platform. Start scenarios from its events, or create contacts, tag them, record events and send transactional email from your other apps. Free, like everything in ReputeCast.

## Connect

1. In ReputeCast, open **Settings › Integrations › Make › Connect**, choose what Make may do, and create the key. ReputeCast shows it once.
2. In Make, create a ReputeCast connection with the **Instance URL** (for example `https://reputecast.acme.io`) and the **Key**.

Your instance must be reachable from the internet: Make calls it, and ReputeCast calls Make's webhooks for the triggers.

## Triggers

Watch Subscribed Contacts, Watch Unsubscribed Contacts, Watch Updated Contacts, Watch Form Submissions (any form, or one), Watch Sent Campaigns and Watch Transactional Bounces. Creating the trigger's webhook registers it in ReputeCast (shown as "Managed by Make"); deleting it removes it.

## Actions

Create or Update a Contact, Add a Contact to a List, Add a Tag to a Contact, Record an Event and Send a Transactional Email.

ReputeCast never adds someone back to a list they left, never adds a suppressed address and doesn't send transactional email to one: those bundles end with a DataError that says why. Add an **Ignore** or **Resume** error handler to the module to skip them and keep the scenario running.
