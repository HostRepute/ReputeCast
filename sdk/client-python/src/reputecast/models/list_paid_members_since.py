from enum import StrEnum

class ListPaidMembersSince(StrEnum):
    VALUE_0 = "7d"
    VALUE_1 = "30d"
    VALUE_2 = "90d"
    VALUE_3 = "1y"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
