from enum import StrEnum

class AutomationTriggerKind(StrEnum):
    ABANDONED_CHECKOUT = "abandoned_checkout"
    API_EVENT = "api_event"
    BOUGHT_PRODUCT = "bought_product"
    DATE_FIELD = "date_field"
    FIRST_ORDER = "first_order"
    FORM_SUBMITTED = "form_submitted"
    JOINED_LIST = "joined_list"
    LINK_CLICKED = "link_clicked"
    MILESTONE_REACHED = "milestone_reached"
    ORDER_PLACED = "order_placed"
    PAID_ENDED = "paid_ended"
    PAID_STARTED = "paid_started"
    PAID_TIER_CHANGED = "paid_tier_changed"
    PAID_TRIAL_ENDING = "paid_trial_ending"
    POLL_ANSWERED = "poll_answered"
    SCORE_CROSSES = "score_crosses"
    SEGMENT_ENTERED = "segment_entered"
    SEGMENT_LEFT = "segment_left"
    TAG_ADDED = "tag_added"

    def __str__(self) -> str:
        return str(self.value)
