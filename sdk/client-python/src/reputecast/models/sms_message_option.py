from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsMessageOption")



@_attrs_define
class SmsMessageOption:
    """ 
        Attributes:
            key (str):
            name (str):
            segments (int):
     """

    key: str
    name: str
    segments: int





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        name = self.name

        segments = self.segments


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "name": name,
            "segments": segments,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        name = d.pop("name")

        segments = d.pop("segments")

        sms_message_option = cls(
            key=key,
            name=name,
            segments=segments,
        )

        return sms_message_option

