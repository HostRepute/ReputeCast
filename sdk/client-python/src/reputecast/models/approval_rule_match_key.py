from enum import StrEnum

class ApprovalRuleMatchKey(StrEnum):
    APPROVAL_LISTS = "approval_lists"
    AUDIENCE_MIN = "audience_min"
    AUTHOR_LACKS_SEND = "author_lacks_send"
    AUTOMATION_PUBLISH = "automation_publish"
    CHANNEL_COST_MIN = "channel_cost_min"
    CLIENT_SIGNOFF = "client_signoff"
    NEW_PROVIDER = "new_provider"
    POST_PUBLISH = "post_publish"
    READY_WITH_WARNINGS = "ready_with_warnings"
    TRANSACTIONAL_PUBLISH = "transactional_publish"
    WARMING_PROVIDER = "warming_provider"

    def __str__(self) -> str:
        return str(self.value)
