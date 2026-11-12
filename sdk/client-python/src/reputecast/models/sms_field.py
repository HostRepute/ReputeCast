from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsField")



@_attrs_define
class SmsField:
    """ 
        Attributes:
            fallback (str):
            has_fallback (bool):
            key (str):
     """

    fallback: str
    has_fallback: bool
    key: str





    def to_dict(self) -> dict[str, Any]:
        fallback = self.fallback

        has_fallback = self.has_fallback

        key = self.key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fallback": fallback,
            "hasFallback": has_fallback,
            "key": key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fallback = d.pop("fallback")

        has_fallback = d.pop("hasFallback")

        key = d.pop("key")

        sms_field = cls(
            fallback=fallback,
            has_fallback=has_fallback,
            key=key,
        )

        return sms_field

