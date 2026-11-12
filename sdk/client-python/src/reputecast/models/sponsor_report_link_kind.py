from enum import StrEnum

class SponsorReportLinkKind(StrEnum):
    BUTTON = "button"
    LINK = "link"
    LOGO = "logo"

    def __str__(self) -> str:
        return str(self.value)
