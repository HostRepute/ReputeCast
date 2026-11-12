from enum import StrEnum

class TransactionalTestRenderSource(StrEnum):
    LAST_CALL = "last_call"
    TEST_DATA = "test_data"

    def __str__(self) -> str:
        return str(self.value)
