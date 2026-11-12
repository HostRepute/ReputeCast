from enum import StrEnum

class SegmentListItemRefresh(StrEnum):
    DYNAMIC = "dynamic"
    STATIC = "static"

    def __str__(self) -> str:
        return str(self.value)
