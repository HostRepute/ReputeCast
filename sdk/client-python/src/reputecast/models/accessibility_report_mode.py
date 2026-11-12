from enum import StrEnum

class AccessibilityReportMode(StrEnum):
    HTML = "html"
    MJML = "mjml"

    def __str__(self) -> str:
        return str(self.value)
