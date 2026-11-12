from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_contact_channels_line_source import SmsContactChannelsLineSource
from ..models.sms_contact_channels_line_type import SmsContactChannelsLineType
from ..models.sms_contact_channels_zone_source import SmsContactChannelsZoneSource
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_consent_event import SmsConsentEvent
  from ..models.sms_contact_list import SmsContactList
  from ..models.sms_suppression import SmsSuppression





T = TypeVar("T", bound="SmsContactChannels")



@_attrs_define
class SmsContactChannels:
    """ 
        Attributes:
            can_edit (bool):
            country (str):
            display (str):
            events (list[SmsConsentEvent]):
            line_source (SmsContactChannelsLineSource):
            line_type (SmsContactChannelsLineType):
            lists (list[SmsContactList]):
            phone (str):
            shared (int):
            sms_set_up (bool):
            time_zone (str):
            whats_app_set_up (bool):
            zone_source (SmsContactChannelsZoneSource):
            suppressed (SmsSuppression | Unset):
     """

    can_edit: bool
    country: str
    display: str
    events: list[SmsConsentEvent]
    line_source: SmsContactChannelsLineSource
    line_type: SmsContactChannelsLineType
    lists: list[SmsContactList]
    phone: str
    shared: int
    sms_set_up: bool
    time_zone: str
    whats_app_set_up: bool
    zone_source: SmsContactChannelsZoneSource
    suppressed: SmsSuppression | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_consent_event import SmsConsentEvent # noqa: PLC0415
        from ..models.sms_contact_list import SmsContactList # noqa: PLC0415
        from ..models.sms_suppression import SmsSuppression # noqa: PLC0415
        can_edit = self.can_edit

        country = self.country

        display = self.display

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        line_source = self.line_source.value

        line_type = self.line_type.value

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        phone = self.phone

        shared = self.shared

        sms_set_up = self.sms_set_up

        time_zone = self.time_zone

        whats_app_set_up = self.whats_app_set_up

        zone_source = self.zone_source.value

        suppressed: dict[str, Any] | Unset = UNSET
        if not isinstance(self.suppressed, Unset):
            suppressed = self.suppressed.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "country": country,
            "display": display,
            "events": events,
            "lineSource": line_source,
            "lineType": line_type,
            "lists": lists,
            "phone": phone,
            "shared": shared,
            "smsSetUp": sms_set_up,
            "timeZone": time_zone,
            "whatsAppSetUp": whats_app_set_up,
            "zoneSource": zone_source,
        })
        if suppressed is not UNSET:
            field_dict["suppressed"] = suppressed

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_consent_event import SmsConsentEvent # noqa: PLC0415
        from ..models.sms_contact_list import SmsContactList # noqa: PLC0415
        from ..models.sms_suppression import SmsSuppression # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        country = d.pop("country")

        display = d.pop("display")

        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = SmsConsentEvent.from_dict(events_item_data)



            events.append(events_item)


        line_source = SmsContactChannelsLineSource(d.pop("lineSource"))




        line_type = SmsContactChannelsLineType(d.pop("lineType"))




        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = SmsContactList.from_dict(lists_item_data)



            lists.append(lists_item)


        phone = d.pop("phone")

        shared = d.pop("shared")

        sms_set_up = d.pop("smsSetUp")

        time_zone = d.pop("timeZone")

        whats_app_set_up = d.pop("whatsAppSetUp")

        zone_source = SmsContactChannelsZoneSource(d.pop("zoneSource"))




        _suppressed = d.pop("suppressed", UNSET)
        suppressed: SmsSuppression | Unset
        if isinstance(_suppressed,  Unset):
            suppressed = UNSET
        else:
            suppressed = SmsSuppression.from_dict(_suppressed)




        sms_contact_channels = cls(
            can_edit=can_edit,
            country=country,
            display=display,
            events=events,
            line_source=line_source,
            line_type=line_type,
            lists=lists,
            phone=phone,
            shared=shared,
            sms_set_up=sms_set_up,
            time_zone=time_zone,
            whats_app_set_up=whats_app_set_up,
            zone_source=zone_source,
            suppressed=suppressed,
        )

        return sms_contact_channels

