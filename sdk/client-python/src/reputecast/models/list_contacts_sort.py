from enum import StrEnum

class ListContactsSort(StrEnum):
    ACTIVITY = "activity"
    EMAIL = "email"
    JOINED = "joined"
    LAST_OPEN = "last_open"
    SCORE = "score"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)
