from enum import StrEnum

class HoldoutResultsWindow(StrEnum):
    PERIOD = "period"
    VALUE_1 = "30d"
    VALUE_2 = "90d"

    def __str__(self) -> str:
        return str(self.value)
