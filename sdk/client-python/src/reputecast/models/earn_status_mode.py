from enum import StrEnum

class EarnStatusMode(StrEnum):
    LIVE = "live"
    TEST = "test"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
