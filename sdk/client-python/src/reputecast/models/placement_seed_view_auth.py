from enum import StrEnum

class PlacementSeedViewAuth(StrEnum):
    OAUTH = "oauth"
    PASSWORD = "password"

    def __str__(self) -> str:
        return str(self.value)
