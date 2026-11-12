from enum import StrEnum

class ReferralPageSettingsShareButtonsItem(StrEnum):
    COPY = "copy"
    EMAIL = "email"
    LINKEDIN = "linkedin"
    WHATSAPP = "whatsapp"
    X = "x"

    def __str__(self) -> str:
        return str(self.value)
