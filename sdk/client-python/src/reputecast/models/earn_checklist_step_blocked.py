from enum import StrEnum

class EarnChecklistStepBlocked(StrEnum):
    NO_ROUTE = "no_route"
    VALUE_1 = ""

    def __str__(self) -> str:
        return str(self.value)
