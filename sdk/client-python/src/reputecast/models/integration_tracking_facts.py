from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="IntegrationTrackingFacts")



@_attrs_define
class IntegrationTrackingFacts:
    """ 
        Attributes:
            events_today (int):
            line (str):
            sites (int):
            last_event_at (datetime.datetime | Unset):
     """

    events_today: int
    line: str
    sites: int
    last_event_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        events_today = self.events_today

        line = self.line

        sites = self.sites

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "eventsToday": events_today,
            "line": line,
            "sites": sites,
        })
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        events_today = d.pop("eventsToday")

        line = d.pop("line")

        sites = d.pop("sites")

        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        integration_tracking_facts = cls(
            events_today=events_today,
            line=line,
            sites=sites,
            last_event_at=last_event_at,
        )

        return integration_tracking_facts

