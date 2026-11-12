from enum import StrEnum

class ProviderRouteShareColor(StrEnum):
    OTHER = "other"
    POSTAL = "postal"
    POSTMARK = "postmark"
    SES = "ses"

    def __str__(self) -> str:
        return str(self.value)
