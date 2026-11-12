from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SuppressionCountsView")



@_attrs_define
class SuppressionCountsView:
    """ 
        Attributes:
            complaint (int):
            hard_bounce (int):
            manual (int):
            other (int):
            total (int):
            unsubscribe (int):
     """

    complaint: int
    hard_bounce: int
    manual: int
    other: int
    total: int
    unsubscribe: int





    def to_dict(self) -> dict[str, Any]:
        complaint = self.complaint

        hard_bounce = self.hard_bounce

        manual = self.manual

        other = self.other

        total = self.total

        unsubscribe = self.unsubscribe


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "complaint": complaint,
            "hardBounce": hard_bounce,
            "manual": manual,
            "other": other,
            "total": total,
            "unsubscribe": unsubscribe,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        complaint = d.pop("complaint")

        hard_bounce = d.pop("hardBounce")

        manual = d.pop("manual")

        other = d.pop("other")

        total = d.pop("total")

        unsubscribe = d.pop("unsubscribe")

        suppression_counts_view = cls(
            complaint=complaint,
            hard_bounce=hard_bounce,
            manual=manual,
            other=other,
            total=total,
            unsubscribe=unsubscribe,
        )

        return suppression_counts_view

