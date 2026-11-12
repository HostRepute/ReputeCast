from enum import StrEnum

class WebsiteHiddenBlockReason(StrEnum):
    CONDITIONAL = "conditional"
    EMAIL_ONLY = "email_only"
    HIDDEN = "hidden"

    def __str__(self) -> str:
        return str(self.value)
