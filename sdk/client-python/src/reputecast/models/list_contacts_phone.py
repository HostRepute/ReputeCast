from enum import StrEnum

class ListContactsPhone(StrEnum):
    NO = "no"
    VALUE_2 = ""
    YES = "yes"

    def __str__(self) -> str:
        return str(self.value)
