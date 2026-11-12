from enum import StrEnum

class WebsiteAccessCampaignVisibility(StrEnum):
    AUDIENCE = "audience"
    MEMBERS = "members"
    PAID = "paid"
    PUBLIC = "public"

    def __str__(self) -> str:
        return str(self.value)
