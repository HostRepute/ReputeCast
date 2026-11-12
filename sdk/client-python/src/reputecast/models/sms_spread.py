from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsSpread")



@_attrs_define
class SmsSpread:
    """ 
        Attributes:
            recipients (int):
            segments (int):
     """

    recipients: int
    segments: int





    def to_dict(self) -> dict[str, Any]:
        recipients = self.recipients

        segments = self.segments


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "recipients": recipients,
            "segments": segments,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        recipients = d.pop("recipients")

        segments = d.pop("segments")

        sms_spread = cls(
            recipients=recipients,
            segments=segments,
        )

        return sms_spread

