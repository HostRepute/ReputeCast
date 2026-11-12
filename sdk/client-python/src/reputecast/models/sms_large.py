from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsLarge")



@_attrs_define
class SmsLarge:
    """ 
        Attributes:
            cost (float):
            recipients (int):
            required (bool):
     """

    cost: float
    recipients: int
    required: bool





    def to_dict(self) -> dict[str, Any]:
        cost = self.cost

        recipients = self.recipients

        required = self.required


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cost": cost,
            "recipients": recipients,
            "required": required,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost = d.pop("cost")

        recipients = d.pop("recipients")

        required = d.pop("required")

        sms_large = cls(
            cost=cost,
            recipients=recipients,
            required=required,
        )

        return sms_large

