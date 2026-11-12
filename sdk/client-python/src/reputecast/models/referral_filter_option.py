from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralFilterOption")



@_attrs_define
class ReferralFilterOption:
    """ 
        Attributes:
            count (int):
            link_id (str):
            name (str):
     """

    count: int
    link_id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        link_id = self.link_id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "linkId": link_id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        link_id = d.pop("linkId")

        name = d.pop("name")

        referral_filter_option = cls(
            count=count,
            link_id=link_id,
            name=name,
        )

        return referral_filter_option

