from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ReviewTyped")



@_attrs_define
class ReviewTyped:
    """ 
        Attributes:
            needed (bool):
            reasons (list[str]):
     """

    needed: bool
    reasons: list[str]





    def to_dict(self) -> dict[str, Any]:
        needed = self.needed

        reasons = self.reasons




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "needed": needed,
            "reasons": reasons,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        needed = d.pop("needed")

        reasons = cast(list[str], d.pop("reasons"))


        review_typed = cls(
            needed=needed,
            reasons=reasons,
        )

        return review_typed

