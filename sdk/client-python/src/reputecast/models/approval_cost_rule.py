from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ApprovalCostRule")



@_attrs_define
class ApprovalCostRule:
    """ 
        Attributes:
            amount (float):
            enabled (bool):
     """

    amount: float
    enabled: bool





    def to_dict(self) -> dict[str, Any]:
        amount = self.amount

        enabled = self.enabled


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amount": amount,
            "enabled": enabled,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount = d.pop("amount")

        enabled = d.pop("enabled")

        approval_cost_rule = cls(
            amount=amount,
            enabled=enabled,
        )

        return approval_cost_rule

