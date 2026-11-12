from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_preview_part_kind import SmsPreviewPartKind






T = TypeVar("T", bound="SmsPreviewPart")



@_attrs_define
class SmsPreviewPart:
    """ 
        Attributes:
            kind (SmsPreviewPartKind):
            text (str):
     """

    kind: SmsPreviewPartKind
    text: str





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = SmsPreviewPartKind(d.pop("kind"))




        text = d.pop("text")

        sms_preview_part = cls(
            kind=kind,
            text=text,
        )

        return sms_preview_part

