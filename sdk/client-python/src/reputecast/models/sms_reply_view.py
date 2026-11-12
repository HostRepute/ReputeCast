from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.count import Count





T = TypeVar("T", bound="SmsReplyView")



@_attrs_define
class SmsReplyView:
    """ 
        Attributes:
            count (Count):
            text (str):
     """

    count: Count
    text: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.count import Count # noqa: PLC0415
        count = self.count.to_dict()

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.count import Count # noqa: PLC0415
        d = dict(src_dict)
        count = Count.from_dict(d.pop("count"))




        text = d.pop("text")

        sms_reply_view = cls(
            count=count,
            text=text,
        )

        return sms_reply_view

