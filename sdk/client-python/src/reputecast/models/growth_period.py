from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.growth_period_kind import GrowthPeriodKind






T = TypeVar("T", bound="GrowthPeriod")



@_attrs_define
class GrowthPeriod:
    """ 
        Attributes:
            days (int):
            from_ (str): YYYY-MM-DD
            kind (GrowthPeriodKind):
            to (str): YYYY-MM-DD
     """

    days: int
    from_: str
    kind: GrowthPeriodKind
    to: str





    def to_dict(self) -> dict[str, Any]:
        days = self.days

        from_ = self.from_

        kind = self.kind.value

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "from": from_,
            "kind": kind,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        days = d.pop("days")

        from_ = d.pop("from")

        kind = GrowthPeriodKind(d.pop("kind"))




        to = d.pop("to")

        growth_period = cls(
            days=days,
            from_=from_,
            kind=kind,
            to=to,
        )

        return growth_period

