from enum import StrEnum

class RecommendationsBlockWhich(StrEnum):
    ALL = "all"
    PICK = "pick"

    def __str__(self) -> str:
        return str(self.value)
