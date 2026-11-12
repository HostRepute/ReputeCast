from enum import StrEnum

class ImportFieldChoiceType(StrEnum):
    BOOLEAN = "boolean"
    DATE = "date"
    NUMBER = "number"
    TEXT = "text"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
