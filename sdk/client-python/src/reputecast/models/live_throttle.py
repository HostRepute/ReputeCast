from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveThrottle")



@_attrs_define
class LiveThrottle:
    """ 
        Attributes:
            mailbox (str):
            rate (float):
     """

    mailbox: str
    rate: float





    def to_dict(self) -> dict[str, Any]:
        mailbox = self.mailbox

        rate = self.rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mailbox": mailbox,
            "rate": rate,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mailbox = d.pop("mailbox")

        rate = d.pop("rate")

        live_throttle = cls(
            mailbox=mailbox,
            rate=rate,
        )

        return live_throttle

