from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WarmupMailboxCap")



@_attrs_define
class WarmupMailboxCap:
    """ 
        Attributes:
            cap (int):
            mailbox (str):
            percent (int):
            sent (int):
     """

    cap: int
    mailbox: str
    percent: int
    sent: int





    def to_dict(self) -> dict[str, Any]:
        cap = self.cap

        mailbox = self.mailbox

        percent = self.percent

        sent = self.sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cap": cap,
            "mailbox": mailbox,
            "percent": percent,
            "sent": sent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cap = d.pop("cap")

        mailbox = d.pop("mailbox")

        percent = d.pop("percent")

        sent = d.pop("sent")

        warmup_mailbox_cap = cls(
            cap=cap,
            mailbox=mailbox,
            percent=percent,
            sent=sent,
        )

        return warmup_mailbox_cap

