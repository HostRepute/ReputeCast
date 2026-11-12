from enum import StrEnum

class EarnProviderCardKey(StrEnum):
    LEMONSQUEEZY = "lemonsqueezy"
    PADDLE = "paddle"
    STRIPE = "stripe"

    def __str__(self) -> str:
        return str(self.value)
