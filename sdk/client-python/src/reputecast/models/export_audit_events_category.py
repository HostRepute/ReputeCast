from enum import StrEnum

class ExportAuditEventsCategory(StrEnum):
    AGENCY = "agency"
    AI = "ai"
    AUDIENCE = "audience"
    COMMERCE = "commerce"
    DIGESTS = "digests"
    EARN = "earn"
    GROW = "grow"
    INTEGRATIONS = "integrations"
    PLACEMENT = "placement"
    POLLS = "polls"
    PRODUCTS = "products"
    PUSH = "push"
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
