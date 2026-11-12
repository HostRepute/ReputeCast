from enum import StrEnum

class CustomFieldInputType(StrEnum):
    BOOLEAN = "boolean"
    DATE = "date"
    NUMBER = "number"
    SELECT = "select"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
