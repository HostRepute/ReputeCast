from enum import StrEnum

class CampaignFollowUpInputKind(StrEnum):
    CLICKERS = "clickers"
    NON_OPENERS = "non_openers"

    def __str__(self) -> str:
        return str(self.value)
