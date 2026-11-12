from enum import StrEnum

class ApprovalListItemSubjectType(StrEnum):
    AUTOMATION = "automation"
    CAMPAIGN = "campaign"
    TRANSACTIONAL_TEMPLATE = "transactional_template"
    WEBSITE_PAGE = "website_page"
    WEBSITE_POST = "website_post"

    def __str__(self) -> str:
        return str(self.value)
