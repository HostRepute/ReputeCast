from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_sender_row_kind import SmsSenderRowKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_registration_view import SmsRegistrationView





T = TypeVar("T", bound="SmsSenderRow")



@_attrs_define
class SmsSenderRow:
    """ 
        Attributes:
            adapter (str):
            can_send (bool):
            countries (list[str]):
            held (bool):
            id (str):
            kind (SmsSenderRowKind):
            label (str):
            manual (bool):
            provider (str):
            provider_id (str):
            registration (SmsRegistrationView):
            routes (list[list[str]]):
            throughput (float):
            value (str):
            brand (SmsRegistrationView | Unset):
            campaign (SmsRegistrationView | Unset):
     """

    adapter: str
    can_send: bool
    countries: list[str]
    held: bool
    id: str
    kind: SmsSenderRowKind
    label: str
    manual: bool
    provider: str
    provider_id: str
    registration: SmsRegistrationView
    routes: list[list[str]]
    throughput: float
    value: str
    brand: SmsRegistrationView | Unset = UNSET
    campaign: SmsRegistrationView | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        adapter = self.adapter

        can_send = self.can_send

        countries = self.countries



        held = self.held

        id = self.id

        kind = self.kind.value

        label = self.label

        manual = self.manual

        provider = self.provider

        provider_id = self.provider_id

        registration = self.registration.to_dict()

        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data


            routes.append(routes_item)



        throughput = self.throughput

        value = self.value

        brand: dict[str, Any] | Unset = UNSET
        if not isinstance(self.brand, Unset):
            brand = self.brand.to_dict()

        campaign: dict[str, Any] | Unset = UNSET
        if not isinstance(self.campaign, Unset):
            campaign = self.campaign.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "canSend": can_send,
            "countries": countries,
            "held": held,
            "id": id,
            "kind": kind,
            "label": label,
            "manual": manual,
            "provider": provider,
            "providerId": provider_id,
            "registration": registration,
            "routes": routes,
            "throughput": throughput,
            "value": value,
        })
        if brand is not UNSET:
            field_dict["brand"] = brand
        if campaign is not UNSET:
            field_dict["campaign"] = campaign

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        can_send = d.pop("canSend")

        countries = cast(list[str], d.pop("countries"))


        held = d.pop("held")

        id = d.pop("id")

        kind = SmsSenderRowKind(d.pop("kind"))




        label = d.pop("label")

        manual = d.pop("manual")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        registration = SmsRegistrationView.from_dict(d.pop("registration"))




        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = cast(list[str], routes_item_data)

            routes.append(routes_item)


        throughput = d.pop("throughput")

        value = d.pop("value")

        _brand = d.pop("brand", UNSET)
        brand: SmsRegistrationView | Unset
        if isinstance(_brand,  Unset):
            brand = UNSET
        else:
            brand = SmsRegistrationView.from_dict(_brand)




        _campaign = d.pop("campaign", UNSET)
        campaign: SmsRegistrationView | Unset
        if isinstance(_campaign,  Unset):
            campaign = UNSET
        else:
            campaign = SmsRegistrationView.from_dict(_campaign)




        sms_sender_row = cls(
            adapter=adapter,
            can_send=can_send,
            countries=countries,
            held=held,
            id=id,
            kind=kind,
            label=label,
            manual=manual,
            provider=provider,
            provider_id=provider_id,
            registration=registration,
            routes=routes,
            throughput=throughput,
            value=value,
            brand=brand,
            campaign=campaign,
        )

        return sms_sender_row

