from enum import StrEnum

class WhatsAppAccountStatus(StrEnum):
    CONNECTED = "connected"
    TOKEN_REJECTED = "token_rejected"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
