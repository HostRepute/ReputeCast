from enum import StrEnum

class ProductPriceKind(StrEnum):
    ONE_TIME = "one_time"
    PWYW = "pwyw"

    def __str__(self) -> str:
        return str(self.value)
