from enum import StrEnum

class SponsorPaymentStatus(StrEnum):
    OVERDUE = "overdue"
    PAID = "paid"
    UNPAID = "unpaid"

    def __str__(self) -> str:
        return str(self.value)
