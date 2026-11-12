from enum import StrEnum

class EarnWebhookStatus(StrEnum):
    MISSING = "missing"
    RECEIVING = "receiving"
    SILENT = "silent"
    UNREGISTERED = "unregistered"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
