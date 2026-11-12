from enum import StrEnum

class ContactFilterSort(StrEnum):
    ACTIVITY = "activity"
    EMAIL = "email"
    JOINED = "joined"
    LAST_OPEN = "last_open"
    SCORE = "score"

    def __str__(self) -> str:
        return str(self.value)
