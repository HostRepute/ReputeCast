from enum import StrEnum

class SegmentDetailRefresh(StrEnum):
    DYNAMIC = "dynamic"
    STATIC = "static"

    def __str__(self) -> str:
        return str(self.value)
