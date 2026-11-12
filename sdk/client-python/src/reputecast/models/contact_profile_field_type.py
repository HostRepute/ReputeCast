from enum import StrEnum

class ContactProfileFieldType(StrEnum):
    BOOLEAN = "boolean"
    BUILTIN = "builtin"
    DATE = "date"
    NUMBER = "number"
    SELECT = "select"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
