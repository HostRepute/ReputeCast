from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LanguageDefaults")



@_attrs_define
class LanguageDefaults:
    """ 
        Attributes:
            default (str):
            fallback (str):
     """

    default: str
    fallback: str





    def to_dict(self) -> dict[str, Any]:
        default = self.default

        fallback = self.fallback


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "default": default,
            "fallback": fallback,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        default = d.pop("default")

        fallback = d.pop("fallback")

        language_defaults = cls(
            default=default,
            fallback=fallback,
        )

        return language_defaults

