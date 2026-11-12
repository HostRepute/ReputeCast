from enum import StrEnum

class RenderService(StrEnum):
    LITMUS = "litmus"
    MAILGUN_INSPECT = "mailgun_inspect"

    def __str__(self) -> str:
        return str(self.value)
