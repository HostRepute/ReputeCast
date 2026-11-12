from enum import StrEnum

class TransactionalCheckWhere(StrEnum):
    BODY = "body"
    PREHEADER = "preheader"
    SUBJECT = "subject"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
