from enum import StrEnum

class DigestFeedBlockOverflow(StrEnum):
    DROP = "drop"
    KEEP = "keep"

    def __str__(self) -> str:
        return str(self.value)
