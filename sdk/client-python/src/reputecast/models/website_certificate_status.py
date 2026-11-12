from enum import StrEnum

class WebsiteCertificateStatus(StrEnum):
    FAILING = "failing"
    NONE = "none"
    PENDING = "pending"
    UNKNOWN = "unknown"
    VALID = "valid"

    def __str__(self) -> str:
        return str(self.value)
