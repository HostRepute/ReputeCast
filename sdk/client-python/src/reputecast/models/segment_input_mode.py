from enum import StrEnum

class SegmentInputMode(StrEnum):
    BUILDER = "builder"
    SQL = "sql"

    def __str__(self) -> str:
        return str(self.value)
