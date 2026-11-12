from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sponsor_booking_event_data import SponsorBookingEventData





T = TypeVar("T", bound="SponsorBookingEvent")



@_attrs_define
class SponsorBookingEvent:
    """ 
        Attributes:
            at (datetime.datetime):
            data (SponsorBookingEventData):
            kind (str):
            actor (str | Unset):
     """

    at: datetime.datetime
    data: SponsorBookingEventData
    kind: str
    actor: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_booking_event_data import SponsorBookingEventData # noqa: PLC0415
        at = self.at.isoformat()

        data = self.data.to_dict()

        kind = self.kind

        actor = self.actor


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "data": data,
            "kind": kind,
        })
        if actor is not UNSET:
            field_dict["actor"] = actor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_booking_event_data import SponsorBookingEventData # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        data = SponsorBookingEventData.from_dict(d.pop("data"))




        kind = d.pop("kind")

        actor = d.pop("actor", UNSET)

        sponsor_booking_event = cls(
            at=at,
            data=data,
            kind=kind,
            actor=actor,
        )

        return sponsor_booking_event

