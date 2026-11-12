from enum import StrEnum

class RouteStreamImpactBasis(StrEnum):
    CONTACTS = "contacts"
    RECENT = "recent"

    def __str__(self) -> str:
        return str(self.value)
