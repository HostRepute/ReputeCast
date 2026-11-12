from enum import StrEnum

class OverviewSenderHealthLabel(StrEnum):
    FAIR = "Fair"
    GOOD = "Good"
    POOR = "Poor"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
