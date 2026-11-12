from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sms_timeline_sample_ev import SmsTimelineSampleEv





T = TypeVar("T", bound="SmsTimelineSample")



@_attrs_define
class SmsTimelineSample:
    """ 
        Attributes:
            contact_id (str):
            events (list[SmsTimelineSampleEv]):
            name (str):
     """

    contact_id: str
    events: list[SmsTimelineSampleEv]
    name: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_timeline_sample_ev import SmsTimelineSampleEv # noqa: PLC0415
        contact_id = self.contact_id

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contactId": contact_id,
            "events": events,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_timeline_sample_ev import SmsTimelineSampleEv # noqa: PLC0415
        d = dict(src_dict)
        contact_id = d.pop("contactId")

        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = SmsTimelineSampleEv.from_dict(events_item_data)



            events.append(events_item)


        name = d.pop("name")

        sms_timeline_sample = cls(
            contact_id=contact_id,
            events=events,
            name=name,
        )

        return sms_timeline_sample

