from enum import StrEnum

class RemovalItemStatus(StrEnum):
    DRAFT = "draft"
    HELD = "held"
    OUTDATED = "outdated"
    UP_TO_DATE = "up_to_date"

    def __str__(self) -> str:
        return str(self.value)
