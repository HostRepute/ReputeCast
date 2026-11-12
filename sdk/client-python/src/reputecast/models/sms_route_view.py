from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_registration_view import SmsRegistrationView





T = TypeVar("T", bound="SmsRouteView")



@_attrs_define
class SmsRouteView:
    """ 
        Attributes:
            adapter (str):
            countries (list[str]):
            id (str):
            on_lists (int):
            provider (str):
            registration (SmsRegistrationView):
            sender (str):
            sender_id (str):
            sender_kind (str):
            skipped (bool):
            fallback (str | Unset):
            fallback_id (str | Unset):
            fallback_provider (str | Unset):
     """

    adapter: str
    countries: list[str]
    id: str
    on_lists: int
    provider: str
    registration: SmsRegistrationView
    sender: str
    sender_id: str
    sender_kind: str
    skipped: bool
    fallback: str | Unset = UNSET
    fallback_id: str | Unset = UNSET
    fallback_provider: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        adapter = self.adapter

        countries = self.countries



        id = self.id

        on_lists = self.on_lists

        provider = self.provider

        registration = self.registration.to_dict()

        sender = self.sender

        sender_id = self.sender_id

        sender_kind = self.sender_kind

        skipped = self.skipped

        fallback = self.fallback

        fallback_id = self.fallback_id

        fallback_provider = self.fallback_provider


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "countries": countries,
            "id": id,
            "onLists": on_lists,
            "provider": provider,
            "registration": registration,
            "sender": sender,
            "senderId": sender_id,
            "senderKind": sender_kind,
            "skipped": skipped,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if fallback_id is not UNSET:
            field_dict["fallbackId"] = fallback_id
        if fallback_provider is not UNSET:
            field_dict["fallbackProvider"] = fallback_provider

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        countries = cast(list[str], d.pop("countries"))


        id = d.pop("id")

        on_lists = d.pop("onLists")

        provider = d.pop("provider")

        registration = SmsRegistrationView.from_dict(d.pop("registration"))




        sender = d.pop("sender")

        sender_id = d.pop("senderId")

        sender_kind = d.pop("senderKind")

        skipped = d.pop("skipped")

        fallback = d.pop("fallback", UNSET)

        fallback_id = d.pop("fallbackId", UNSET)

        fallback_provider = d.pop("fallbackProvider", UNSET)

        sms_route_view = cls(
            adapter=adapter,
            countries=countries,
            id=id,
            on_lists=on_lists,
            provider=provider,
            registration=registration,
            sender=sender,
            sender_id=sender_id,
            sender_kind=sender_kind,
            skipped=skipped,
            fallback=fallback,
            fallback_id=fallback_id,
            fallback_provider=fallback_provider,
        )

        return sms_route_view

