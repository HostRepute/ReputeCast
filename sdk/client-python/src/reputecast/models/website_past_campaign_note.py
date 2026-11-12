from enum import StrEnum

class WebsitePastCampaignNote(StrEnum):
    ARCHIVE_HIDDEN = "archive_hidden"
    LIST_PRIVATE = "list_private"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
