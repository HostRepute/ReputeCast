from enum import StrEnum

class ReferralChannelChannel(StrEnum):
    COPY = "copy"
    EMAIL = "email"
    LINKEDIN = "linkedin"
    OTHER = "other"
    WHATSAPP = "whatsapp"
    X = "x"

    def __str__(self) -> str:
        return str(self.value)
