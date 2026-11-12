from enum import StrEnum

class ComposerPreviewRequestSample(StrEnum):
    CONTACT = "contact"
    NO_FIRST_NAME = "no_first_name"
    RANDOM = "random"

    def __str__(self) -> str:
        return str(self.value)
