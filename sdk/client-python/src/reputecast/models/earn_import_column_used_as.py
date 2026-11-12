from enum import StrEnum

class EarnImportColumnUsedAs(StrEnum):
    CUSTOMER = "customer"
    EMAIL = "email"
    NAME = "name"
    PAID_THROUGH = "paid_through"
    PLAN = "plan"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)
