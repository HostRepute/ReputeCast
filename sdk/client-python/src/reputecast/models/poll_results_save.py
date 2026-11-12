from enum import StrEnum

class PollResultsSave(StrEnum):
    FIELD = "field"
    NONE = "none"
    TAG = "tag"

    def __str__(self) -> str:
        return str(self.value)
