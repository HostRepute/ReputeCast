from enum import StrEnum

class RouteStreamImpactStream(StrEnum):
    AUTOMATION = "automation"
    SYSTEM = "system"
    TRANSACTIONAL = "transactional"

    def __str__(self) -> str:
        return str(self.value)
