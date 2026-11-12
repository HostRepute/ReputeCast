from enum import StrEnum

class RecommendationsBlockLayout(StrEnum):
    COLUMNS = "columns"
    COMPACT = "compact"
    STACKED = "stacked"

    def __str__(self) -> str:
        return str(self.value)
