from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.approval_whats_app_value import ApprovalWhatsAppValue





T = TypeVar("T", bound="ApprovalWhatsAppSnapshot")



@_attrs_define
class ApprovalWhatsAppSnapshot:
    """ 
        Attributes:
            body (str):
            business (str):
            buttons (list[str]):
            category (str):
            cost (float):
            language (str):
            number (str):
            preview (str):
            recipients (int):
            schedule (str):
            template (str):
            variables (list[ApprovalWhatsAppValue]):
     """

    body: str
    business: str
    buttons: list[str]
    category: str
    cost: float
    language: str
    number: str
    preview: str
    recipients: int
    schedule: str
    template: str
    variables: list[ApprovalWhatsAppValue]





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_whats_app_value import ApprovalWhatsAppValue # noqa: PLC0415
        body = self.body

        business = self.business

        buttons = self.buttons



        category = self.category

        cost = self.cost

        language = self.language

        number = self.number

        preview = self.preview

        recipients = self.recipients

        schedule = self.schedule

        template = self.template

        variables = []
        for variables_item_data in self.variables:
            variables_item = variables_item_data.to_dict()
            variables.append(variables_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "business": business,
            "buttons": buttons,
            "category": category,
            "cost": cost,
            "language": language,
            "number": number,
            "preview": preview,
            "recipients": recipients,
            "schedule": schedule,
            "template": template,
            "variables": variables,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_whats_app_value import ApprovalWhatsAppValue # noqa: PLC0415
        d = dict(src_dict)
        body = d.pop("body")

        business = d.pop("business")

        buttons = cast(list[str], d.pop("buttons"))


        category = d.pop("category")

        cost = d.pop("cost")

        language = d.pop("language")

        number = d.pop("number")

        preview = d.pop("preview")

        recipients = d.pop("recipients")

        schedule = d.pop("schedule")

        template = d.pop("template")

        variables = []
        _variables = d.pop("variables")
        for variables_item_data in (_variables):
            variables_item = ApprovalWhatsAppValue.from_dict(variables_item_data)



            variables.append(variables_item)


        approval_whats_app_snapshot = cls(
            body=body,
            business=business,
            buttons=buttons,
            category=category,
            cost=cost,
            language=language,
            number=number,
            preview=preview,
            recipients=recipients,
            schedule=schedule,
            template=template,
            variables=variables,
        )

        return approval_whats_app_snapshot

