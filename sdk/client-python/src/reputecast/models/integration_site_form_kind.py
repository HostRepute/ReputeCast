from enum import StrEnum

class IntegrationSiteFormKind(StrEnum):
    BLOCK = "block"
    SHORTCODE = "shortcode"
    WIDGET = "widget"

    def __str__(self) -> str:
        return str(self.value)
