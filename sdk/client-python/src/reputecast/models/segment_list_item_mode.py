from enum import StrEnum

class SegmentListItemMode(StrEnum):
    BUILDER = "builder"
    SQL = "sql"

    def __str__(self) -> str:
        return str(self.value)
