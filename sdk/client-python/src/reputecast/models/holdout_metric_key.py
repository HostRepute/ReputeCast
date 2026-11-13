from enum import StrEnum

class HoldoutMetricKey(StrEnum):
    CONVERSION_RATE = "conversion_rate"
    ORDERS_PER_CONTACT = "orders_per_contact"
    REVENUE_PER_CONTACT = "revenue_per_contact"
    SITE_VISITORS = "site_visitors"

    def __str__(self) -> str:
        return str(self.value)
