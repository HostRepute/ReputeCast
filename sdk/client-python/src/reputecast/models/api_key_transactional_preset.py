from enum import StrEnum

class APIKeyTransactionalPreset(StrEnum):
    SEND_ONLY = "send_only"
    SMTP = "smtp"

    def __str__(self) -> str:
        return str(self.value)
