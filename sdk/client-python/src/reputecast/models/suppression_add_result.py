from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="SuppressionAddResult")



@_attrs_define
class SuppressionAddResult:
    """ 
        Attributes:
            added (int):
            already (int): Already on the list
            invalid (list[str]): Values that aren't addresses
     """

    added: int
    already: int
    invalid: list[str]





    def to_dict(self) -> dict[str, Any]:
        added = self.added

        already = self.already

        invalid = self.invalid




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "added": added,
            "already": already,
            "invalid": invalid,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        added = d.pop("added")

        already = d.pop("already")

        invalid = cast(list[str], d.pop("invalid"))


        suppression_add_result = cls(
            added=added,
            already=already,
            invalid=invalid,
        )

        return suppression_add_result

