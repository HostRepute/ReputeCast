from enum import StrEnum

class SmsPauseViewKind(StrEnum):
    AUTO = "auto"
    BUDGET = "budget"
    OUTAGE = "outage"
    PERSON = "person"

    def __str__(self) -> str:
        return str(self.value)
