from enum import IntEnum

class GiftPageInputLinkMonths(IntEnum):
    VALUE_3 = 3
    VALUE_6 = 6
    VALUE_12 = 12

    def __str__(self) -> str:
        return str(self.value)
