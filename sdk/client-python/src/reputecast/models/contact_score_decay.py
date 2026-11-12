from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ContactScoreDecay")



@_attrs_define
class ContactScoreDecay:
    """ 
        Attributes:
            enabled (bool):
            lost (int):
            quiet_weeks (int):
            last_activity (datetime.datetime | Unset):
            last_activity_kind (str | Unset): open or click
            last_open (datetime.datetime | Unset):
     """

    enabled: bool
    lost: int
    quiet_weeks: int
    last_activity: datetime.datetime | Unset = UNSET
    last_activity_kind: str | Unset = UNSET
    last_open: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        enabled = self.enabled

        lost = self.lost

        quiet_weeks = self.quiet_weeks

        last_activity: str | Unset = UNSET
        if not isinstance(self.last_activity, Unset):
            last_activity = self.last_activity.isoformat()

        last_activity_kind = self.last_activity_kind

        last_open: str | Unset = UNSET
        if not isinstance(self.last_open, Unset):
            last_open = self.last_open.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "lost": lost,
            "quietWeeks": quiet_weeks,
        })
        if last_activity is not UNSET:
            field_dict["lastActivity"] = last_activity
        if last_activity_kind is not UNSET:
            field_dict["lastActivityKind"] = last_activity_kind
        if last_open is not UNSET:
            field_dict["lastOpen"] = last_open

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        enabled = d.pop("enabled")

        lost = d.pop("lost")

        quiet_weeks = d.pop("quietWeeks")

        _last_activity = d.pop("lastActivity", UNSET)
        last_activity: datetime.datetime | Unset
        if isinstance(_last_activity,  Unset):
            last_activity = UNSET
        else:
            last_activity = datetime.datetime.fromisoformat(_last_activity)




        last_activity_kind = d.pop("lastActivityKind", UNSET)

        _last_open = d.pop("lastOpen", UNSET)
        last_open: datetime.datetime | Unset
        if isinstance(_last_open,  Unset):
            last_open = UNSET
        else:
            last_open = datetime.datetime.fromisoformat(_last_open)




        contact_score_decay = cls(
            enabled=enabled,
            lost=lost,
            quiet_weeks=quiet_weeks,
            last_activity=last_activity,
            last_activity_kind=last_activity_kind,
            last_open=last_open,
        )

        return contact_score_decay

