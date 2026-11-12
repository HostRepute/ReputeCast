from enum import StrEnum

class ComposerTestSendSample(StrEnum):
    CONTACT = "contact"
    NO_FIRST_NAME = "no_first_name"

    def __str__(self) -> str:
        return str(self.value)
