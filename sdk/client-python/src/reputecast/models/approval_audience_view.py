from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ApprovalAudienceView")



@_attrs_define
class ApprovalAudienceView:
    """ 
        Attributes:
            change_pct (float):
            count (int):
            current (int):
            excluded (str):
            lists (list[str]):
     """

    change_pct: float
    count: int
    current: int
    excluded: str
    lists: list[str]





    def to_dict(self) -> dict[str, Any]:
        change_pct = self.change_pct

        count = self.count

        current = self.current

        excluded = self.excluded

        lists = self.lists




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changePct": change_pct,
            "count": count,
            "current": current,
            "excluded": excluded,
            "lists": lists,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        change_pct = d.pop("changePct")

        count = d.pop("count")

        current = d.pop("current")

        excluded = d.pop("excluded")

        lists = cast(list[str], d.pop("lists"))


        approval_audience_view = cls(
            change_pct=change_pct,
            count=count,
            current=current,
            excluded=excluded,
            lists=lists,
        )

        return approval_audience_view

