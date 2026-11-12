from enum import StrEnum

class TransactionalSendingKeyPreset(StrEnum):
    SEND_ONLY = "send_only"
    SMTP = "smtp"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
