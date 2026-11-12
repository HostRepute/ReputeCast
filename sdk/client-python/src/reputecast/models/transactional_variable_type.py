from enum import StrEnum

class TransactionalVariableType(StrEnum):
    BOOLEAN = "boolean"
    DATE = "date"
    LIST = "list"
    NUMBER = "number"
    OBJECT = "object"
    TEXT = "text"
    URL = "url"

    def __str__(self) -> str:
        return str(self.value)
