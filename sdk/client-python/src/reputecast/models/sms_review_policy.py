from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsReviewPolicy")



@_attrs_define
class SmsReviewPolicy:
    """ 
        Attributes:
            audience_min (int):
            audience_rule (bool):
            author_lacks_send (bool):
            cost_amount (float):
            cost_rule (bool):
     """

    audience_min: int
    audience_rule: bool
    author_lacks_send: bool
    cost_amount: float
    cost_rule: bool





    def to_dict(self) -> dict[str, Any]:
        audience_min = self.audience_min

        audience_rule = self.audience_rule

        author_lacks_send = self.author_lacks_send

        cost_amount = self.cost_amount

        cost_rule = self.cost_rule


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audienceMin": audience_min,
            "audienceRule": audience_rule,
            "authorLacksSend": author_lacks_send,
            "costAmount": cost_amount,
            "costRule": cost_rule,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        audience_min = d.pop("audienceMin")

        audience_rule = d.pop("audienceRule")

        author_lacks_send = d.pop("authorLacksSend")

        cost_amount = d.pop("costAmount")

        cost_rule = d.pop("costRule")

        sms_review_policy = cls(
            audience_min=audience_min,
            audience_rule=audience_rule,
            author_lacks_send=author_lacks_send,
            cost_amount=cost_amount,
            cost_rule=cost_rule,
        )

        return sms_review_policy

