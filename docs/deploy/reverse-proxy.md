# Reverse proxy and TLS

ReputeCast listens on plain HTTP (`:8080`). In production, put it behind a
proxy that terminates TLS, such as Caddy, nginx or Traefik.

The public URL has to be right, and reachable from the internet:
- sending providers deliver bounce and complaint webhooks to
  `<public URL>/hooks/…`;
- links in emails, public pages, passkeys and SSO redirects use it.

When a provider can't reach it, the connect wizard and the first-run setup say
so; they check DNS, TLS and reachability for you.

## Settings

| Setting | Value |
|---|---|
| `REPUTECAST_PUBLIC_URL` | The https address the proxy serves, for example `https://news.example.com`. |
| `REPUTECAST_HTTP__TRUSTED_PROXIES` | The proxy's address or network, for example `127.0.0.1` on the same host, or `172.16.0.0/12` for a proxy in another Docker container. Only then does ReputeCast read the client IP from `X-Forwarded-For`. Rate limits and the audit log use that IP. |

HTTPS-only cookies and HSTS follow the public URL. Restart ReputeCast after
changing either setting.

## Caddy

Caddy gets and renews the certificate itself, and speaks HTTP/2 and HTTP/3
to browsers without being asked:

```caddyfile
news.example.com {
	reverse_proxy localhost:8080
}
```

## nginx

```nginx
server {
    listen 443 ssl;
    # The web app loads a screen's code in many small files: over HTTP/2 a
    # browser asks for them all at once, over HTTP/1.1 six at a time. On a
    # slow phone network that is about 6 s or 12 s before Contacts shows.
    # (Before nginx 1.25.1: `listen 443 ssl http2;` instead.)
    http2 on;
    server_name news.example.com;

    ssl_certificate     /etc/letsencrypt/live/news.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/news.example.com/privkey.pem;

    # Imports upload files of up to 105 MB (contact lists, transactional
    # template archives); product files go up in 4 MB chunks.
    client_max_body_size 110m;

    # /metrics lives on its own port (9464); never proxy that one. The main
    # listener answers /metrics with 404 unless a metrics token is set.
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # The live send monitor streams server-sent events.
        proxy_buffering off;
        proxy_read_timeout 1h;
    }
}
```

## Testing without a public server

On a laptop or a staging box, expose port 8080 with a tunnel (Cloudflare
Tunnel or ngrok) and use the tunnel's https address as the public URL. Paste
it into the connect wizard when it asks.

## Custom domains for public pages

A workspace can serve its subscribe page, preference center, unsubscribe page
and archive at the root of its own domain, for example
`https://news.acme.io/subscribe`. Emails keep linking to the public URL.

1. Enter the domain in Forms & public pages › Branding.
2. Point it at the server with a `CNAME` to the public URL's host. For an apex
   domain, use `A`/`AAAA` records with the same addresses, or the
   `_reputecast.<domain>` TXT record shown there.
3. Give the proxy a certificate for the domain, and keep the original `Host`
   header. The pages answer on the domain once the DNS check passes.

A `CNAME` or matching addresses only show that a domain points at this
server, not which workspace it belongs to. So a domain another workspace
already uses (for its pages, its SMS short links or its link tracking) can't
be added, and a domain under the public URL's own host (such as
`acme.reputecast.example.com` beside `reputecast.example.com`, where a
wildcard record points every name at the server) verifies only with its TXT
record, whose value is different for each workspace.

The same goes for an SMS short-link domain, checked again every day: a domain
that another workspace stored and verified first (before this rule existed),
or one under the public URL's own host, needs the `_reputecast.<domain>` TXT
record shown in Routing & rules besides its `CNAME`. Until then texts with
links don't use it, and a short-link domain only ever carries its own
workspace's links. A link tracking domain, checked every 30 minutes, works
the same way, with the TXT record on its sending domain's records page;
until then links and the open pixel use the public URL. Nobody recorded when
a tracking domain was verified before this rule, so one that two workspaces
verified then needs the TXT record of whichever keeps it.

With Caddy's on-demand TLS, certificates are issued only for domains that
ReputeCast has verified (`/p/_/tls-check` answers 200 for them):

```caddyfile
{
	on_demand_tls {
		ask http://localhost:8080/p/_/tls-check
	}
}

news.example.com {
	reverse_proxy localhost:8080
}

https:// {
	tls {
		on_demand
	}
	reverse_proxy localhost:8080
}
```
