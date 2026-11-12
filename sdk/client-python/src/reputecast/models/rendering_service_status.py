from enum import StrEnum

class RenderingServiceStatus(StrEnum):
    CONNECTED = "connected"
    FAILING = "failing"

    def __str__(self) -> str:
        return str(self.value)
