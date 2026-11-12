from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ContactProfileStats")



@_attrs_define
class ContactProfileStats:
    """ 
        Attributes:
            clicks (int):
            emails90d (int):
            human_opens (int): Emails opened by a person
            last_open_machine (bool):
            human_open_rate (float | Unset):
            last_open_at (datetime.datetime | Unset):
     """

    clicks: int
    emails90d: int
    human_opens: int
    last_open_machine: bool
    human_open_rate: float | Unset = UNSET
    last_open_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        emails90d = self.emails90d

        human_opens = self.human_opens

        last_open_machine = self.last_open_machine

        human_open_rate = self.human_open_rate

        last_open_at: str | Unset = UNSET
        if not isinstance(self.last_open_at, Unset):
            last_open_at = self.last_open_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "emails90d": emails90d,
            "humanOpens": human_opens,
            "lastOpenMachine": last_open_machine,
        })
        if human_open_rate is not UNSET:
            field_dict["humanOpenRate"] = human_open_rate
        if last_open_at is not UNSET:
            field_dict["lastOpenAt"] = last_open_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        emails90d = d.pop("emails90d")

        human_opens = d.pop("humanOpens")

        last_open_machine = d.pop("lastOpenMachine")

        human_open_rate = d.pop("humanOpenRate", UNSET)

        _last_open_at = d.pop("lastOpenAt", UNSET)
        last_open_at: datetime.datetime | Unset
        if isinstance(_last_open_at,  Unset):
            last_open_at = UNSET
        else:
            last_open_at = datetime.datetime.fromisoformat(_last_open_at)




        contact_profile_stats = cls(
            clicks=clicks,
            emails90d=emails90d,
            human_opens=human_opens,
            last_open_machine=last_open_machine,
            human_open_rate=human_open_rate,
            last_open_at=last_open_at,
        )

        return contact_profile_stats

