from enum import StrEnum

class RenderingConnectInputService(StrEnum):
    LITMUS = "litmus"
    MAILGUN_INSPECT = "mailgun_inspect"

    def __str__(self) -> str:
        return str(self.value)
