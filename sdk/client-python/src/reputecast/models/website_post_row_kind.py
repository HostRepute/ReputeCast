from enum import StrEnum

class WebsitePostRowKind(StrEnum):
    CAMPAIGN = "campaign"
    WEB = "web"

    def __str__(self) -> str:
        return str(self.value)
