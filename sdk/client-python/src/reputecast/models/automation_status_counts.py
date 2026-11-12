from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationStatusCounts")



@_attrs_define
class AutomationStatusCounts:
    """ 
        Attributes:
            all_ (int):
            drafts (int):
            live (int):
            paused (int):
     """

    all_: int
    drafts: int
    live: int
    paused: int





    def to_dict(self) -> dict[str, Any]:
        all_ = self.all_

        drafts = self.drafts

        live = self.live

        paused = self.paused


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "drafts": drafts,
            "live": live,
            "paused": paused,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        all_ = d.pop("all")

        drafts = d.pop("drafts")

        live = d.pop("live")

        paused = d.pop("paused")

        automation_status_counts = cls(
            all_=all_,
            drafts=drafts,
            live=live,
            paused=paused,
        )

        return automation_status_counts

