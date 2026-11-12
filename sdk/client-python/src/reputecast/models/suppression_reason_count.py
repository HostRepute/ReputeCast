from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SuppressionReasonCount")



@_attrs_define
class SuppressionReasonCount:
    """ 
        Attributes:
            count (int):
            reason (str):
     """

    count: int
    reason: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        reason = d.pop("reason")

        suppression_reason_count = cls(
            count=count,
            reason=reason,
        )

        return suppression_reason_count

