from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ApplyResult")



@_attrs_define
class ApplyResult:
    """ 
        Attributes:
            languages (list[str]):
            replaced (int):
     """

    languages: list[str]
    replaced: int





    def to_dict(self) -> dict[str, Any]:
        languages = self.languages



        replaced = self.replaced


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "languages": languages,
            "replaced": replaced,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        languages = cast(list[str], d.pop("languages"))


        replaced = d.pop("replaced")

        apply_result = cls(
            languages=languages,
            replaced=replaced,
        )

        return apply_result

