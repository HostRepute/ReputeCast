from enum import StrEnum

class RenderingConnectInputRegion(StrEnum):
    EU = "eu"
    US = "us"

    def __str__(self) -> str:
        return str(self.value)
