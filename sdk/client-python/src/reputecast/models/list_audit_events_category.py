from enum import StrEnum

class ListAuditEventsCategory(StrEnum):
    AGENCY = "agency"
    AI = "ai"
    AUDIENCE = "audience"
    COMMERCE = "commerce"
    COUPONS = "coupons"
    DIGESTS = "digests"
    EARN = "earn"
    GROW = "grow"
    INTEGRATIONS = "integrations"
    LEADS = "leads"
    PLACEMENT = "placement"
    POLLS = "polls"
    PRODUCTS = "products"
    PUSH = "push"
    REPLIES = "replies"
    ROUTING = "routing"
    SCORING = "scoring"
    SENDING = "sending"
    SETTINGS = "settings"
    SMS = "sms"
    TEAM = "team"
    TRANSACTIONAL = "transactional"
    VERIFICATION = "verification"
    WEBSITE = "website"
    WHATSAPP = "whatsapp"

    def __str__(self) -> str:
        return str(self.value)
