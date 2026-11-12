from enum import StrEnum

class DigestIssueInfoHoldReason(StrEnum):
    CHANGES = "changes"
    NO_APPROVERS = "no_approvers"
    READINESS = "readiness"
    REJECTED = "rejected"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
