from enum import StrEnum

class PublicTurnstileStatus(StrEnum):
    ERROR = "error"
    OFF = "off"
    OK = "ok"

    def __str__(self) -> str:
        return str(self.value)
