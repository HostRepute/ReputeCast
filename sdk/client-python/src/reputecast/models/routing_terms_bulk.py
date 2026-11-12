from enum import StrEnum

class RoutingTermsBulk(StrEnum):
    ALLOWED = "allowed"
    BROADCAST_STREAMS_ONLY = "broadcast_streams_only"
    FORBIDDEN = "forbidden"
    YOUR_POLICY = "your_policy"

    def __str__(self) -> str:
        return str(self.value)
