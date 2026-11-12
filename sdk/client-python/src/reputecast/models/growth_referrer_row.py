from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GrowthReferrerRow")



@_attrs_define
class GrowthReferrerRow:
    """ 
        Attributes:
            joins (int):
            page (str):
     """

    joins: int
    page: str





    def to_dict(self) -> dict[str, Any]:
        joins = self.joins

        page = self.page


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "joins": joins,
            "page": page,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        joins = d.pop("joins")

        page = d.pop("page")

        growth_referrer_row = cls(
            joins=joins,
            page=page,
        )

        return growth_referrer_row

