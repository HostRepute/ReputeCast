from enum import StrEnum

class CampaignBulkInputAction(StrEnum):
    ARCHIVE = "archive"
    DELETE = "delete"
    TAG = "tag"
    UNARCHIVE = "unarchive"
    UNTAG = "untag"

    def __str__(self) -> str:
        return str(self.value)
