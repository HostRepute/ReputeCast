from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_route_test_route import SmsRouteTestRoute
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_registration_view import SmsRegistrationView





T = TypeVar("T", bound="SmsRouteTest")



@_attrs_define
class SmsRouteTest:
    """ 
        Attributes:
            adapter (str):
            calling_code (str):
            can_reply (bool):
            can_send (bool):
            cost (float):
            country (str):
            inside (bool):
            local (str):
            number (str):
            opens (str):
            price (float):
            provider (str):
            registration (SmsRegistrationView):
            route (SmsRouteTestRoute):
            route_countries (list[str]):
            segments (int):
            sender (str):
            sender_kind (str):
            window (str):
            zone (str):
            zone_source (str):
            help_number (str | Unset):
            suppressed (str | Unset):
     """

    adapter: str
    calling_code: str
    can_reply: bool
    can_send: bool
    cost: float
    country: str
    inside: bool
    local: str
    number: str
    opens: str
    price: float
    provider: str
    registration: SmsRegistrationView
    route: SmsRouteTestRoute
    route_countries: list[str]
    segments: int
    sender: str
    sender_kind: str
    window: str
    zone: str
    zone_source: str
    help_number: str | Unset = UNSET
    suppressed: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        adapter = self.adapter

        calling_code = self.calling_code

        can_reply = self.can_reply

        can_send = self.can_send

        cost = self.cost

        country = self.country

        inside = self.inside

        local = self.local

        number = self.number

        opens = self.opens

        price = self.price

        provider = self.provider

        registration = self.registration.to_dict()

        route = self.route.value

        route_countries = self.route_countries



        segments = self.segments

        sender = self.sender

        sender_kind = self.sender_kind

        window = self.window

        zone = self.zone

        zone_source = self.zone_source

        help_number = self.help_number

        suppressed = self.suppressed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "callingCode": calling_code,
            "canReply": can_reply,
            "canSend": can_send,
            "cost": cost,
            "country": country,
            "inside": inside,
            "local": local,
            "number": number,
            "opens": opens,
            "price": price,
            "provider": provider,
            "registration": registration,
            "route": route,
            "routeCountries": route_countries,
            "segments": segments,
            "sender": sender,
            "senderKind": sender_kind,
            "window": window,
            "zone": zone,
            "zoneSource": zone_source,
        })
        if help_number is not UNSET:
            field_dict["helpNumber"] = help_number
        if suppressed is not UNSET:
            field_dict["suppressed"] = suppressed

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        calling_code = d.pop("callingCode")

        can_reply = d.pop("canReply")

        can_send = d.pop("canSend")

        cost = d.pop("cost")

        country = d.pop("country")

        inside = d.pop("inside")

        local = d.pop("local")

        number = d.pop("number")

        opens = d.pop("opens")

        price = d.pop("price")

        provider = d.pop("provider")

        registration = SmsRegistrationView.from_dict(d.pop("registration"))




        route = SmsRouteTestRoute(d.pop("route"))




        route_countries = cast(list[str], d.pop("routeCountries"))


        segments = d.pop("segments")

        sender = d.pop("sender")

        sender_kind = d.pop("senderKind")

        window = d.pop("window")

        zone = d.pop("zone")

        zone_source = d.pop("zoneSource")

        help_number = d.pop("helpNumber", UNSET)

        suppressed = d.pop("suppressed", UNSET)

        sms_route_test = cls(
            adapter=adapter,
            calling_code=calling_code,
            can_reply=can_reply,
            can_send=can_send,
            cost=cost,
            country=country,
            inside=inside,
            local=local,
            number=number,
            opens=opens,
            price=price,
            provider=provider,
            registration=registration,
            route=route,
            route_countries=route_countries,
            segments=segments,
            sender=sender,
            sender_kind=sender_kind,
            window=window,
            zone=zone,
            zone_source=zone_source,
            help_number=help_number,
            suppressed=suppressed,
        )

        return sms_route_test

