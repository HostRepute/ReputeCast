from enum import StrEnum

class ReportLinkRowPlace(StrEnum):
    BODY = "body"
    FOOTER = "footer"
    HEADER = "header"

    def __str__(self) -> str:
        return str(self.value)
