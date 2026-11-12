from enum import StrEnum

class SponsorBlockLabelMode(StrEnum):
    NAME = "name"
    PLAIN = "plain"

    def __str__(self) -> str:
        return str(self.value)
