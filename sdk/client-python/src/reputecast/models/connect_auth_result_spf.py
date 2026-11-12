from enum import StrEnum

class ConnectAuthResultSpf(StrEnum):
    FAIL = "fail"
    NONE = "none"
    PASS = "pass"

    def __str__(self) -> str:
        return str(self.value)
