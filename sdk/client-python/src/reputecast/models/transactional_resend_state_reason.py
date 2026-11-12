from enum import StrEnum

class TransactionalResendStateReason(StrEnum):
    ATTACHMENTS = "attachments"
    NOT_SENT = "not_sent"
    PERMISSION = "permission"
    RAW = "raw"
    SENSITIVE = "sensitive"
    TEMPLATE = "template"
    VALUE_6 = ""

    def __str__(self) -> str:
        return str(self.value)
