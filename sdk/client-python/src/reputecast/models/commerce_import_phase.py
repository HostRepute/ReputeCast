from enum import StrEnum

class CommerceImportPhase(StrEnum):
    CUSTOMERS = "customers"
    DONE = "done"
    MATCHING = "matching"
    ORDERS = "orders"
    PRODUCTS = "products"

    def __str__(self) -> str:
        return str(self.value)
