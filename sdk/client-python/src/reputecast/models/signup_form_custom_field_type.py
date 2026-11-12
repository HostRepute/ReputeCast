from enum import StrEnum

class SignupFormCustomFieldType(StrEnum):
    BOOLEAN = "boolean"
    DATE = "date"
    NUMBER = "number"
    SELECT = "select"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
