from enum import StrEnum

class ListSuppressionsSince(StrEnum):
    VALUE_0 = "7d"
    VALUE_1 = "30d"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
