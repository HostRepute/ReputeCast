from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_contact_card_last_method import WebsiteContactCardLastMethod
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_member_event import WebsiteMemberEvent
  from ..models.website_meter_use import WebsiteMeterUse





T = TypeVar("T", bound="WebsiteContactCard")



@_attrs_define
class WebsiteContactCard:
    """ 
        Attributes:
            can_sign_out (bool):
            events (list[WebsiteMemberEvent]):
            host (str):
            member (bool):
            paid (bool):
            sessions (int):
            sign_ins (int):
            site_on (bool):
            tiers (list[str]):
            last_method (WebsiteContactCardLastMethod | Unset):
            last_sign_in (datetime.datetime | Unset):
            meter (WebsiteMeterUse | Unset):
            reason (str | Unset):
            session_ends (datetime.datetime | Unset):
            since (datetime.datetime | Unset):
     """

    can_sign_out: bool
    events: list[WebsiteMemberEvent]
    host: str
    member: bool
    paid: bool
    sessions: int
    sign_ins: int
    site_on: bool
    tiers: list[str]
    last_method: WebsiteContactCardLastMethod | Unset = UNSET
    last_sign_in: datetime.datetime | Unset = UNSET
    meter: WebsiteMeterUse | Unset = UNSET
    reason: str | Unset = UNSET
    session_ends: datetime.datetime | Unset = UNSET
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_member_event import WebsiteMemberEvent # noqa: PLC0415
        from ..models.website_meter_use import WebsiteMeterUse # noqa: PLC0415
        can_sign_out = self.can_sign_out

        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        host = self.host

        member = self.member

        paid = self.paid

        sessions = self.sessions

        sign_ins = self.sign_ins

        site_on = self.site_on

        tiers = self.tiers



        last_method: str | Unset = UNSET
        if not isinstance(self.last_method, Unset):
            last_method = self.last_method.value


        last_sign_in: str | Unset = UNSET
        if not isinstance(self.last_sign_in, Unset):
            last_sign_in = self.last_sign_in.isoformat()

        meter: dict[str, Any] | Unset = UNSET
        if not isinstance(self.meter, Unset):
            meter = self.meter.to_dict()

        reason = self.reason

        session_ends: str | Unset = UNSET
        if not isinstance(self.session_ends, Unset):
            session_ends = self.session_ends.isoformat()

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canSignOut": can_sign_out,
            "events": events,
            "host": host,
            "member": member,
            "paid": paid,
            "sessions": sessions,
            "signIns": sign_ins,
            "siteOn": site_on,
            "tiers": tiers,
        })
        if last_method is not UNSET:
            field_dict["lastMethod"] = last_method
        if last_sign_in is not UNSET:
            field_dict["lastSignIn"] = last_sign_in
        if meter is not UNSET:
            field_dict["meter"] = meter
        if reason is not UNSET:
            field_dict["reason"] = reason
        if session_ends is not UNSET:
            field_dict["sessionEnds"] = session_ends
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_member_event import WebsiteMemberEvent # noqa: PLC0415
        from ..models.website_meter_use import WebsiteMeterUse # noqa: PLC0415
        d = dict(src_dict)
        can_sign_out = d.pop("canSignOut")

        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = WebsiteMemberEvent.from_dict(events_item_data)



            events.append(events_item)


        host = d.pop("host")

        member = d.pop("member")

        paid = d.pop("paid")

        sessions = d.pop("sessions")

        sign_ins = d.pop("signIns")

        site_on = d.pop("siteOn")

        tiers = cast(list[str], d.pop("tiers"))


        _last_method = d.pop("lastMethod", UNSET)
        last_method: WebsiteContactCardLastMethod | Unset
        if isinstance(_last_method,  Unset):
            last_method = UNSET
        else:
            last_method = WebsiteContactCardLastMethod(_last_method)




        _last_sign_in = d.pop("lastSignIn", UNSET)
        last_sign_in: datetime.datetime | Unset
        if isinstance(_last_sign_in,  Unset):
            last_sign_in = UNSET
        else:
            last_sign_in = datetime.datetime.fromisoformat(_last_sign_in)




        _meter = d.pop("meter", UNSET)
        meter: WebsiteMeterUse | Unset
        if isinstance(_meter,  Unset):
            meter = UNSET
        else:
            meter = WebsiteMeterUse.from_dict(_meter)




        reason = d.pop("reason", UNSET)

        _session_ends = d.pop("sessionEnds", UNSET)
        session_ends: datetime.datetime | Unset
        if isinstance(_session_ends,  Unset):
            session_ends = UNSET
        else:
            session_ends = datetime.datetime.fromisoformat(_session_ends)




        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        website_contact_card = cls(
            can_sign_out=can_sign_out,
            events=events,
            host=host,
            member=member,
            paid=paid,
            sessions=sessions,
            sign_ins=sign_ins,
            site_on=site_on,
            tiers=tiers,
            last_method=last_method,
            last_sign_in=last_sign_in,
            meter=meter,
            reason=reason,
            session_ends=session_ends,
            since=since,
        )

        return website_contact_card

