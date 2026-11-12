from enum import StrEnum

class DigestFeedStatusStatus(StrEnum):
    INVALID = "invalid"
    OK = "ok"
    UNREACHABLE = "unreachable"
    UNREAD = "unread"

    def __str__(self) -> str:
        return str(self.value)
