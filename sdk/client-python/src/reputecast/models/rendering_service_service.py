from enum import StrEnum

class RenderingServiceService(StrEnum):
    LITMUS = "litmus"
    MAILGUN_INSPECT = "mailgun_inspect"

    def __str__(self) -> str:
        return str(self.value)
