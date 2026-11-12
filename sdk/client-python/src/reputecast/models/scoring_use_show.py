from enum import StrEnum

class ScoringUseShow(StrEnum):
    EXCEPT = "except"
    ONLY = "only"

    def __str__(self) -> str:
        return str(self.value)
