from enum import StrEnum

class WebsitePaymentsMode(StrEnum):
    LIVE = "live"
    TEST = "test"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
