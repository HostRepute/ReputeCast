from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsMessageEvent")



@_attrs_define
class SmsMessageEvent:
    """ 
        Attributes:
            at (datetime.datetime):
            kind (str):
            code (str | Unset):
     """

    at: datetime.datetime
    kind: str
    code: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        kind = self.kind

        code = self.code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "kind": kind,
        })
        if code is not UNSET:
            field_dict["code"] = code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        kind = d.pop("kind")

        code = d.pop("code", UNSET)

        sms_message_event = cls(
            at=at,
            kind=kind,
            code=code,
        )

        return sms_message_event

