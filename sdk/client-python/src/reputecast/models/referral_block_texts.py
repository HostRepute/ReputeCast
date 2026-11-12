from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralBlockTexts")



@_attrs_define
class ReferralBlockTexts:
    """ 
        Attributes:
            many (str):
            none (str):
            one (str):
     """

    many: str
    none: str
    one: str





    def to_dict(self) -> dict[str, Any]:
        many = self.many

        none = self.none

        one = self.one


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "many": many,
            "none": none,
            "one": one,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        many = d.pop("many")

        none = d.pop("none")

        one = d.pop("one")

        referral_block_texts = cls(
            many=many,
            none=none,
            one=one,
        )

        return referral_block_texts

