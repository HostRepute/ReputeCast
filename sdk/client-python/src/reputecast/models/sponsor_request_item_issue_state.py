from enum import StrEnum

class SponsorRequestItemIssueState(StrEnum):
    HELD = "held"
    OPEN = "open"

    def __str__(self) -> str:
        return str(self.value)
