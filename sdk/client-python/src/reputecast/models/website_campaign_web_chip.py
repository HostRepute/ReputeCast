from enum import StrEnum

class WebsiteCampaignWebChip(StrEnum):
    OFF = "off"
    PUBLISHED = "published"
    SCHEDULED = "scheduled"
    UNPUBLISHED = "unpublished"
    WITH_SEND = "with_send"

    def __str__(self) -> str:
        return str(self.value)
