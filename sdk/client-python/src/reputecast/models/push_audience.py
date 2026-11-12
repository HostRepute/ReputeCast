from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushAudience")



@_attrs_define
class PushAudience:
    """ 
        Attributes:
            anonymous (int):
            final (int):
            linked (int):
            narrowed (bool):
            over_cap (int):
            total (int):
     """

    anonymous: int
    final: int
    linked: int
    narrowed: bool
    over_cap: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        anonymous = self.anonymous

        final = self.final

        linked = self.linked

        narrowed = self.narrowed

        over_cap = self.over_cap

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "anonymous": anonymous,
            "final": final,
            "linked": linked,
            "narrowed": narrowed,
            "overCap": over_cap,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        anonymous = d.pop("anonymous")

        final = d.pop("final")

        linked = d.pop("linked")

        narrowed = d.pop("narrowed")

        over_cap = d.pop("overCap")

        total = d.pop("total")

        push_audience = cls(
            anonymous=anonymous,
            final=final,
            linked=linked,
            narrowed=narrowed,
            over_cap=over_cap,
            total=total,
        )

        return push_audience

