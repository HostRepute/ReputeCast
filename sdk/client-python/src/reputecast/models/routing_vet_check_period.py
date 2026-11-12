from enum import StrEnum

class RoutingVetCheckPeriod(StrEnum):
    DAY = "day"
    MONTH = "month"

    def __str__(self) -> str:
        return str(self.value)
