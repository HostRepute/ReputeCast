from enum import StrEnum

class SurveySaveTo(StrEnum):
    FIELD = "field"
    NONE = "none"
    TAGS = "tags"

    def __str__(self) -> str:
        return str(self.value)
