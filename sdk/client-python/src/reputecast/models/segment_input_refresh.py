from enum import StrEnum

class SegmentInputRefresh(StrEnum):
    DYNAMIC = "dynamic"
    STATIC = "static"

    def __str__(self) -> str:
        return str(self.value)
