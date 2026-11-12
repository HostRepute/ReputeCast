from enum import StrEnum

class SegmentCatalogFieldKind(StrEnum):
    BOOLEAN = "boolean"
    CAMPAIGN = "campaign"
    DATE = "date"
    ENUM = "enum"
    LIST = "list"
    NUMBER = "number"
    PERIOD = "period"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
