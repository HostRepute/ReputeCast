from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WhatsAppFallback")



@_attrs_define
class WhatsAppFallback:
    """ 
        Attributes:
            fallback (str):
            field (str):
            key (str):
     """

    fallback: str
    field: str
    key: str





    def to_dict(self) -> dict[str, Any]:
        fallback = self.fallback

        field = self.field

        key = self.key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fallback": fallback,
            "field": field,
            "key": key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fallback = d.pop("fallback")

        field = d.pop("field")

        key = d.pop("key")

        whats_app_fallback = cls(
            fallback=fallback,
            field=field,
            key=key,
        )

        return whats_app_fallback

