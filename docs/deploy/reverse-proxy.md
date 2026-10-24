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
| `REPUTECAST_HTTP__PUBLIC_URL` (`REPUTECAST_PUBLIC_URL` in the compose `.env`) | The https address the proxy serves, for example `https://news.example.com`. |
| `REPUTECAST_HTTP__TRUSTED_PROXIES` | The proxy's address or network, for example `127.0.0.1` on the same host, or `172.16.0.0/12` for a proxy in another Docker container. Only then does ReputeCast read the client IP from `X-Forwarded-For`. Rate limits and the audit log use that IP. |

HTTPS-only cookies and HSTS follow the public URL. Restart ReputeCast after
changing either setting.

## Caddy

Caddy gets and renews the certificate itself:

```caddyfile
news.example.com {
	reverse_proxy localhost:8080
}
```

## nginx

```nginx
server {
    listen 443 ssl;
    server_name news.example.com;

    ssl_certificate     /etc/letsencrypt/live/news.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/news.example.com/privkey.pem;

    # Imports upload files of up to 105 MB (contact lists, transactional
    # template archives); product files go up in 4 MB chunks.
    client_max_body_size 110m;

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
