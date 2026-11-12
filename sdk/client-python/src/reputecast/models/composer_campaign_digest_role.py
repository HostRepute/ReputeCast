from enum import StrEnum

class ComposerCampaignDigestRole(StrEnum):
    ISSUE = "issue"
    NEXT = "next"
    TEMPLATE = "template"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
