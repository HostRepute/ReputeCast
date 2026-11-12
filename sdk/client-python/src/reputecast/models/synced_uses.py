from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SyncedUses")



@_attrs_define
class SyncedUses:
    """ 
        Attributes:
            drafts (int):
            held (int):
            scheduled (int):
            sent (int):
     """

    drafts: int
    held: int
    scheduled: int
    sent: int





    def to_dict(self) -> dict[str, Any]:
        drafts = self.drafts

        held = self.held

        scheduled = self.scheduled

        sent = self.sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "drafts": drafts,
            "held": held,
            "scheduled": scheduled,
            "sent": sent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        drafts = d.pop("drafts")

        held = d.pop("held")

        scheduled = d.pop("scheduled")

        sent = d.pop("sent")

        synced_uses = cls(
            drafts=drafts,
            held=held,
            scheduled=scheduled,
            sent=sent,
        )

        return synced_uses

