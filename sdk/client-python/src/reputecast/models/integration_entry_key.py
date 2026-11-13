from enum import StrEnum

class IntegrationEntryKey(StrEnum):
    ANTHROPIC = "anthropic"
    API = "api"
    AZURE = "azure"
    BOUNCER = "bouncer"
    COMPATIBLE = "compatible"
    EMAILABLE = "emailable"
    GOOGLE = "google"
    HOSTREPUTE = "hostrepute"
    KICKBOX = "kickbox"
    LEMONSQUEEZY = "lemonsqueezy"
    MAKE = "make"
    MCP = "mcp"
    META_LEAD_ADS = "meta_lead_ads"
    N8N = "n8n"
    NEVERBOUNCE = "neverbounce"
    OPENAI = "openai"
    PADDLE = "paddle"
    SHOPIFY = "shopify"
    STRIPE = "stripe"
    WEBHOOKS = "webhooks"
    WOOCOMMERCE = "woocommerce"
    WORDPRESS = "wordpress"
    ZAPIER = "zapier"
    ZEROBOUNCE = "zerobounce"

    def __str__(self) -> str:
        return str(self.value)
