from enum import StrEnum

class TransactionalRejectionReason(StrEnum):
    OPTED_OUT = "opted_out"
    PROVIDER = "provider"
    SUPPRESSED = "suppressed"
    TEST_NOT_ALLOWED = "test_not_allowed"

    def __str__(self) -> str:
        return str(self.value)
