from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="SmsPreviewBubble")



@_attrs_define
class SmsPreviewBubble:
    """ 
        Attributes:
            at (datetime.datetime):
            text (str):
     """

    at: datetime.datetime
    text: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        text = d.pop("text")

        sms_preview_bubble = cls(
            at=at,
            text=text,
        )

        return sms_preview_bubble

