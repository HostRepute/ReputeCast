from enum import StrEnum

class DeliverabilitySNDSFilterResult(StrEnum):
    GREEN = "green"
    RED = "red"
    VALUE_3 = ""
    YELLOW = "yellow"

    def __str__(self) -> str:
        return str(self.value)
