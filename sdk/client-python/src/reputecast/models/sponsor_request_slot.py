from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sponsor_request_issue import SponsorRequestIssue





T = TypeVar("T", bound="SponsorRequestSlot")



@_attrs_define
class SponsorRequestSlot:
    """ 
        Attributes:
            creative_days (int):
            currency (str):
            id (str):
            issues (list[SponsorRequestIssue]):
            list_name (str):
            name (str):
            pay_days (int):
            price_cents (int):
     """

    creative_days: int
    currency: str
    id: str
    issues: list[SponsorRequestIssue]
    list_name: str
    name: str
    pay_days: int
    price_cents: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_request_issue import SponsorRequestIssue # noqa: PLC0415
        creative_days = self.creative_days

        currency = self.currency

        id = self.id

        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        list_name = self.list_name

        name = self.name

        pay_days = self.pay_days

        price_cents = self.price_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "creativeDays": creative_days,
            "currency": currency,
            "id": id,
            "issues": issues,
            "listName": list_name,
            "name": name,
            "payDays": pay_days,
            "priceCents": price_cents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_request_issue import SponsorRequestIssue # noqa: PLC0415
        d = dict(src_dict)
        creative_days = d.pop("creativeDays")

        currency = d.pop("currency")

        id = d.pop("id")

        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = SponsorRequestIssue.from_dict(issues_item_data)



            issues.append(issues_item)


        list_name = d.pop("listName")

        name = d.pop("name")

        pay_days = d.pop("payDays")

        price_cents = d.pop("priceCents")

        sponsor_request_slot = cls(
            creative_days=creative_days,
            currency=currency,
            id=id,
            issues=issues,
            list_name=list_name,
            name=name,
            pay_days=pay_days,
            price_cents=price_cents,
        )

        return sponsor_request_slot

