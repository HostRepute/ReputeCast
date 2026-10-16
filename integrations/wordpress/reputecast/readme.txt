=== ReputeCast ===
Contributors: hostrepute
Tags: newsletter, signup form, email marketing, subscribe, double opt-in
Requires at least: 6.4
Tested up to: 7.1
Requires PHP: 8.1
Stable tag: 1.3.0
License: GPLv2 or later
License URI: https://www.gnu.org/licenses/gpl-2.0.html

Signup forms from ReputeCast, the open-source newsletter platform, as a block and a shortcode. Published posts can become campaign drafts there.

== Description ==

ReputeCast is an open-source, self-hosted newsletter platform. This plugin connects a WordPress site to a ReputeCast workspace.

* **Signup forms.** Add a form with the "ReputeCast signup form" block, or the shortcode `[reputecast_form id="frm_8c2e1a"]`. The block lists the workspace's forms. Forms load through their own script, as ReputeCast's embed code does: changes made in ReputeCast show on the site, and forms keep working after the site is disconnected.
* **Allowed domains.** A form only takes signups from its allowed domains. The block warns you when this site isn't one of them.
* **Post drafts.** Each post you publish goes to ReputeCast, where the site's post-draft rule decides whether it becomes a campaign draft (by category, with an audience, a template and a sender). Drafts only: nobody gets anything until someone reviews and sends it in ReputeCast.
* **Languages.** With WPML or Polylang, a post's translations become the draft's language versions, and the block has a language setting ("Match the page" by default).
* **Health.** The site reports in when it connects, every 12 hours and with every post: plugin, WordPress and PHP versions, the feed URL, the post categories and where forms are used. ReputeCast's "Check now" calls the site's status endpoint, which answers only calls signed with the site's key.

Everything is free. ReputeCast is licensed under the AGPL; this plugin under the GPL.

== Installation ==

1. Install and activate the plugin: Plugins › Add New, search for "ReputeCast".
2. In ReputeCast, open Settings › Integrations › WordPress and choose **Connect a site**. Copy the site's key: it's shown once.
3. In WordPress, open Settings › ReputeCast, paste your ReputeCast address and the key, then press **Connect**.
4. ReputeCast lists the site once it reports in, a few seconds later.

To add a form, insert the "ReputeCast signup form" block and choose a form, or paste the form's shortcode.

== Frequently Asked Questions ==

= Where do I get ReputeCast? =

ReputeCast runs on your own server. See https://github.com/hostrepute/reputecast to install it.

= The block says the site isn't an allowed domain. =

A form only takes signups from its allowed domains. In ReputeCast, open Forms, choose the form and add this site's address to its allowed domains.

= Does the plugin send my subscribers or visitors anywhere? =

No. Signup forms send their signups straight from the visitor's browser to ReputeCast, as embedded forms do. The plugin itself sends only the site's details and published posts (see External services).

= What happens when I disconnect? =

Signup forms keep working. Posts stop going to ReputeCast, and the site stops reporting in. Disconnect the site in ReputeCast too to revoke its key.

= Does it work with WPML and Polylang? =

Yes. A post's translations go with it and become the draft's language versions, and the form block's language setting lists the site's languages.

== External services ==

The plugin connects to one service: the ReputeCast instance you connect it to, at the address you paste in Settings › ReputeCast (for example https://reputecast.example.com). ReputeCast is self-hosted: you or your organization run it, and its terms and privacy policy are your own.

* **When you press Connect,** the plugin sends the site's key to check it, then reports in.
* **When it reports in** (on Connect, every 12 hours and with every post), it sends the site's address and name, the plugin, WordPress and PHP versions, the feed URL, the post categories and where signup forms are used (each page's title and address).
* **When a post is published,** it sends the post's title, excerpt, address, featured image address, categories and language, and those of its translations.
* **When an editor opens the block,** it fetches the workspace's signup forms: names, statuses and whether this site is an allowed domain.
* **On the pages that show a form,** the visitor's browser loads the form's script from the ReputeCast instance and sends signups there. The script sets no cookies.

Nothing about visitors or users is sent by the plugin. ReputeCast's source code: https://github.com/hostrepute/reputecast

== Changelog ==

= 1.3.0 =
* First release from the ReputeCast repository.
* The "ReputeCast signup form" block and the `[reputecast_form]` shortcode, with the allowed-domain warning.
* Settings › ReputeCast: connect, report in, change the key, disconnect.
* Report-in every 12 hours and with every post, and the signed status endpoint for "Check now".
* Published posts go to ReputeCast's post-draft rule.
* WPML and Polylang: post translations as the draft's language versions, and the block's language setting.

== Upgrade Notice ==

= 1.3.0 =
Post translations (WPML or Polylang) become the draft's language versions, and the form block gets a language setting.
