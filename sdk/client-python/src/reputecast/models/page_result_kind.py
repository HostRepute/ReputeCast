from enum import StrEnum

class PageResultKind(StrEnum):
    NONE = "none"
    REGISTRATIONS = "registrations"
    SALES = "sales"
    SIGNUPS = "signups"

    def __str__(self) -> str:
        return str(self.value)
