from enum import StrEnum

class WordPressCheckOutcome(StrEnum):
    ANSWERED = "answered"
    BLOCKED = "blocked"
    STATUS = "status"
    TIMEOUT = "timeout"
    UNREACHABLE = "unreachable"

    def __str__(self) -> str:
        return str(self.value)
