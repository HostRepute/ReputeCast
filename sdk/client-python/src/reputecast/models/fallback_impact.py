from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="FallbackImpact")



@_attrs_define
class FallbackImpact:
    """ 
        Attributes:
            fallback (str):
            moved (int):
            no_language (int):
            not_offered (int):
     """

    fallback: str
    moved: int
    no_language: int
    not_offered: int





    def to_dict(self) -> dict[str, Any]:
        fallback = self.fallback

        moved = self.moved

        no_language = self.no_language

        not_offered = self.not_offered


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fallback": fallback,
            "moved": moved,
            "noLanguage": no_language,
            "notOffered": not_offered,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fallback = d.pop("fallback")

        moved = d.pop("moved")

        no_language = d.pop("noLanguage")

        not_offered = d.pop("notOffered")

        fallback_impact = cls(
            fallback=fallback,
            moved=moved,
            no_language=no_language,
            not_offered=not_offered,
        )

        return fallback_impact

