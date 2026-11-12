from enum import StrEnum

class TransactionalCheckGroup(StrEnum):
    ACCESSIBILITY = "Accessibility"
    COMPLIANCE = "Compliance"
    CONTENT = "Content"
    LINKS = "Links"
    PRIVACY = "Privacy"
    SUBJECT = "Subject"
    TEST_DATA = "Test data"
    VARIABLES = "Variables"

    def __str__(self) -> str:
        return str(self.value)
