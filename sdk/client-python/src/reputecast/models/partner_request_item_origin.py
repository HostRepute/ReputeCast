from enum import StrEnum

class PartnerRequestItemOrigin(StrEnum):
    CONFIRM = "confirm"
    PAGE = "page"
    PARTNER = "partner"

    def __str__(self) -> str:
        return str(self.value)
