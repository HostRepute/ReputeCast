from enum import StrEnum

class ListHealthRowHealthTone(StrEnum):
    BAD = "bad"
    OK = "ok"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
