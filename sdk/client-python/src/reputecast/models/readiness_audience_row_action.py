from enum import StrEnum

class ReadinessAudienceRowAction(StrEnum):
    INCLUDE_UNENGAGED = "include_unengaged"

    def __str__(self) -> str:
        return str(self.value)
